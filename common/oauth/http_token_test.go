// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

type testTransport struct {
	t       *testing.T
	respMap map[string]string
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tt.t.Helper()
	reqBody, err := io.ReadAll(req.Body)
	if err != nil {
		tt.t.Errorf("error reading request body: %v", err)
	}
	resp := tt.respMap[string(reqBody)]
	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader([]byte(resp))),
		StatusCode: http.StatusOK,
	}, nil
}

func TestHTTPTokenSourceToken(t *testing.T) {
	tests := []struct {
		name       string
		cachedTok  *oauth2.Token
		httpResp   string
		wantAT     string
		wantAT2    string
		wantExpiry int
		wantErr    bool
	}{
		{
			name:       "simple",
			httpResp:   `{"access_token": "at", "expires_in": 10}`,
			wantAT:     "at",
			wantAT2:    "at",
			wantExpiry: 10,
		},
		{
			name:       "get_from_cache",
			cachedTok:  &oauth2.Token{AccessToken: "cached-at", Expiry: time.Now().Add(70 * time.Second)},
			httpResp:   `{"access_token": "at", "expires_in": 10}`,
			wantAT:     "cached-at",
			wantAT2:    "at",
			wantExpiry: 70,
		},
		{
			name:     "error_bad_json",
			httpResp: `{"access_tokn": "at", "expires_in": 10}`,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := `{"grant_type":"client_credentials"}`
			ots, err := newHTTPTokenSource(&configpb.HTTPRequest{
				Data: []string{param},
			}, 0, nil)
			if err != nil {
				t.Errorf("error creating httpTokenSource: %v", err)
			}
			ts := ots.(*httpTokenSource)

			ts.httpClient = &http.Client{
				Transport: &testTransport{
					t:       t,
					respMap: map[string]string{param: tt.httpResp},
				},
			}
			ts.cache.tok = tt.cachedTok

			got, err := ts.Token()
			if (err != nil) != tt.wantErr {
				t.Errorf("httpTokenSource.Token() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			assert.Equal(t, tt.wantAT, got.AccessToken, "access token")
			assert.LessOrEqual(t, got.Expiry, time.Now().Add(time.Duration(tt.wantExpiry)*time.Second), "expiry")

			// Mark the token as expired and try again.
			ts.cache.mu.Lock()
			ts.cache.tok.Expiry = time.Now().Add(-1 * time.Second)
			ts.cache.mu.Unlock()

			got, err = ts.Token()
			if (err != nil) != tt.wantErr {
				t.Errorf("httpTokenSource.Token() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantAT2, got.AccessToken, "access token")
			assert.LessOrEqual(t, got.Expiry, time.Now().Add(time.Duration(tt.wantExpiry)*time.Second), "expiry")
		})
	}
}

func TestNewHTTPTokenSource(t *testing.T) {
	testRefreshExpiryBuffer := 10 * time.Second

	ts, _ := newHTTPTokenSource(&configpb.HTTPRequest{}, testRefreshExpiryBuffer, nil)
	tc := ts.(*httpTokenSource).cache

	assert.Equal(t, testRefreshExpiryBuffer, tc.refreshExpiryBuffer, "token cache refresh expiry buffer")
	assert.Equal(t, tc.ignoreExpiryIfZero, false)
}

func TestRedact(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   `{"access_token": "e1213adada145", "expires_in": 600}`,
			want: `{"access_token": "e1 ........ , "expires_in": 600}`,
		},
		{
			in:   `{"access_token": "e1213adada1"}`,
			want: `{"access_token": "e1213adada1"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := redact(tt.in); got != tt.want {
				t.Errorf("redact() = %v, want %v", got, tt.want)
			}
		})
	}
}
