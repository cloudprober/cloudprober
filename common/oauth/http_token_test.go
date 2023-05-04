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

func TestHTTPTokenSourceToken(t *testing.T) {
	tests := []struct {
		name       string
		cachedTok  *oauth2.Token
		httpResp   string
		wantAT     string
		wantExpiry int
		wantErr    bool
	}{
		{
			name:       "simple",
			httpResp:   `{"access_token": "at", "expires_in": 10}`,
			wantAT:     "at",
			wantExpiry: 10,
		},
		{
			name:       "get_from_cache",
			cachedTok:  &oauth2.Token{AccessToken: "at2", Expiry: time.Now().Add(70 * time.Second)},
			httpResp:   `{"access_token": "at", "expires_in": 10}`,
			wantAT:     "at2",
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
			ts := &httpTokenSource{
				httpDo: func(*http.Request) (*http.Response, error) {
					return &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(tt.httpResp))), StatusCode: http.StatusOK}, nil
				},
			}
			ts.cache = &tokenCache{
				tok:      tt.cachedTok,
				getToken: func() (*oauth2.Token, error) { return ts.tokenFromHTTP(ts.req) },
			}

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
