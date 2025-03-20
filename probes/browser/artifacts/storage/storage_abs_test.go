// Copyright 2025 The Cloudprober Authors.
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

package storage

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStringToSign(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentLength  int64
		accountName    string
		path           string
		expectedString string
	}{
		{
			name:           "Basic PUT request",
			method:         "PUT",
			contentLength:  123,
			accountName:    "test-account",
			path:           "/test-container/test-path",
			expectedString: "PUT\n\n\n123\n\n\n\n\n\n\n\n\nx-ms-blob-type:BlockBlob\nx-ms-date:Thu, 01 Jan 1970 00:00:00 GMT\nx-ms-version:2020-08-04\n/test-account/test-container/test-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs := &ABS{
				accountName: tt.accountName,
			}

			req, err := http.NewRequest(tt.method, "https://example.com"+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.ContentLength = tt.contentLength
			req.Header.Set("x-ms-blob-type", "BlockBlob")
			req.Header.Set("x-ms-date", time.Unix(0, 0).UTC().Format(http.TimeFormat))
			req.Header.Set("x-ms-version", version)

			stringToSign := abs.stringToSign(req)
			assert.Equal(t, tt.expectedString, stringToSign)
		})
	}
}

func TestUploadRequest(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		relPath     string
		expectedURL string
	}{
		{
			name:        "Basic upload request",
			content:     []byte("test content"),
			relPath:     "test-path",
			expectedURL: "https://test-account.blob.core.windows.net/test-container/test-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs := &ABS{
				container:   "test-container",
				accountName: "test-account",
				endpoint:    "https://test-account.blob.core.windows.net",
			}

			req, err := abs.uploadRequest(context.Background(), tt.content, tt.relPath)
			if err != nil {
				t.Fatalf("Failed to create upload request: %v", err)
			}

			assert.Equal(t, tt.expectedURL, req.URL.String())
			assert.Equal(t, "PUT", req.Method)
			assert.Equal(t, int64(len(tt.content)), req.ContentLength)
			assert.Equal(t, "BlockBlob", req.Header.Get("x-ms-blob-type"))
			assert.Equal(t, version, req.Header.Get("x-ms-version"))
			assert.NotEmpty(t, req.Header.Get("x-ms-date"))
		})
	}
}
