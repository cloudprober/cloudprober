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

func TestHTTPTokenSource_Token(t *testing.T) {
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
				tok: tt.cachedTok,
				httpDo: func(*http.Request) (*http.Response, error) {
					return &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(tt.httpResp))), StatusCode: http.StatusOK}, nil
				},
			}
			got, err := ts.Token()
			if (err != nil) != tt.wantErr {
				t.Errorf("jwtTokenSource.Token() error = %v, wantErr %v", err, tt.wantErr)
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
	tests := []struct {
		name        string
		data        []string
		contentType string
		wantReqBody string
		wantCT      string
		wantErr     bool
	}{
		{
			name:        "json_body",
			data:        []string{`{"clientId":"testID", "clientSecret":"testSecret"}`},
			wantReqBody: `{"clientId":"testID", "clientSecret":"testSecret"}`,
			wantCT:      "application/json",
		},
		{
			name:        "query_body",
			data:        []string{"clientId=testID", "clientSecret=testSecret"},
			wantReqBody: "clientId=testID&clientSecret=testSecret",
			wantCT:      "application/x-www-form-urlencoded",
		},
		{
			name:        "explicit_header_override",
			data:        []string{"clientId=testID", "clientSecret=testSecret"},
			contentType: "form-data",
			wantReqBody: "clientId=testID&clientSecret=testSecret",
			wantCT:      "form-data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &configpb.HTTPRequest{}
			c.Data = tt.data

			if tt.contentType != "" {
				c.Header = map[string]string{
					"Content-Type": tt.contentType,
				}
			}

			ts_, err := newHTTPTokenSource(c, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("newHTTPTokenSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			ts := ts_.(*httpTokenSource)
			got, _ := io.ReadAll(ts.req.Body)
			assert.Equal(t, tt.wantReqBody, string(got))

			assert.Equal(t, tt.wantCT, ts.req.Header.Get("Content-Type"), "Content-Type Header")
		})
	}
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
