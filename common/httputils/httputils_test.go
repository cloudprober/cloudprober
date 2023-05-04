// Copyright 2022-2023 The Cloudprober Authors.
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

package httputils

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHandled(t *testing.T) {
	srvMux := http.NewServeMux()

	srvMux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {})
	srvMux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {})
	srvMux.Handle("/", http.RedirectHandler("/status", http.StatusFound))

	tests := map[string]bool{
		"/":            true,
		"/probestatus": false,
		"/status":      true,
		"/config":      true,
		"/config2":     false,
	}

	for url, wantResult := range tests {
		assert.Equal(t, wantResult, IsHandled(srvMux, url))
	}
}

func TestHTTPRequest(t *testing.T) {
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
			headers := map[string]string{}
			if tt.contentType != "" {
				headers["Content-Type"] = tt.contentType
			}

			req, err := HTTPRequest("method", "test-url", tt.data, headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("newHTTPTokenSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, _ := io.ReadAll(req.Body)
			assert.Equal(t, tt.wantReqBody, string(got))
			assert.Equal(t, tt.wantCT, req.Header.Get("Content-Type"), "Content-Type Header")

			got, _ = io.ReadAll(req.Body)
			assert.Equal(t, tt.wantReqBody, string(got))
		})
	}
}
