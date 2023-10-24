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

package notifier

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	httpreqpb "github.com/cloudprober/cloudprober/internal/httpreq/proto"
	"github.com/stretchr/testify/assert"
)

func TestNotifierHTTPNotify(t *testing.T) {
	oldDoHTTPRequest := doHTTPRequest
	defer func() { doHTTPRequest = oldDoHTTPRequest }()

	fields := map[string]string{"alert": "alert1", "target": "target1"}

	tests := []struct {
		name         string
		httpNotifier *httpreqpb.HTTPRequest

		wantMethod   string
		wantURL      string
		wantHeaders  map[string]string
		wantFormData string

		resp    *http.Response
		wantErr string
	}{
		{
			name: "no_error",
			httpNotifier: &httpreqpb.HTTPRequest{
				Method: httpreqpb.HTTPRequest_POST,
				Url:    "http://ntfy.sh/alerttopic_@alert1@_@unmatched@",
				Header: map[string]string{
					"X-Alerting-Target": "@target@",
				},
				Data: []string{"routing_key=testkey", "message=@alert@ for @target@"},
			},
			wantURL: "http://ntfy.sh/alerttopic_@alert1@_@unmatched@",
			wantHeaders: map[string]string{
				"X-Alerting-Target": "target1",
			},
			wantMethod:   "POST",
			wantFormData: "routing_key=testkey&message=alert1%20for%20target1",
		},
		{
			name: "error",
			httpNotifier: &httpreqpb.HTTPRequest{
				Url: "http://ntfy.sh/alerttopic",
			},
			resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("feel like erring out")),
			},
			wantMethod: "GET",
			wantURL:    "http://ntfy.sh/alerttopic",
			wantErr:    "feel like erring out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Notifier{
				httpNotifier: tt.httpNotifier,
			}
			if tt.resp == nil {
				tt.resp = &http.Response{StatusCode: http.StatusOK}
			}

			// Add default headers
			if tt.wantHeaders == nil {
				tt.wantHeaders = make(map[string]string)
			}
			if tt.wantFormData != "" {
				tt.wantHeaders["Content-Type"] = "application/x-www-form-urlencoded"
			}

			doHTTPRequest = func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, tt.wantMethod, req.Method, "Method mismatch")

				assert.Equal(t, tt.wantURL, req.URL.String(), "URL mismatch")

				gotHeaders := make(map[string]string)
				for k, v := range req.Header {
					gotHeaders[k] = v[0]
				}
				assert.Equal(t, tt.wantHeaders, gotHeaders, "Headers mismatch")

				req.ParseForm()
				urlValues, _ := url.ParseQuery(tt.wantFormData)
				assert.Equal(t, urlValues, req.Form, "Data mismatch")
				return tt.resp, nil
			}

			err := n.httpNotify(context.Background(), fields)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
