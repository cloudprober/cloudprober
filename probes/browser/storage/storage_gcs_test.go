// Copyright 2024 The Cloudprober Authors.
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
	"bytes"
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/stretchr/testify/assert"
)

func TestGCSBaseURL(t *testing.T) {
	tests := []struct {
		endpoint string
		bucket   string
		want     string
	}{
		{
			bucket: "cloudprober-browser",
			want:   "https://storage.googleapis.com/upload/storage/v1/b/cloudprober-browser/o?uploadType=media&name=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			cfg := &configpb.GCS{
				Bucket: &tt.bucket,
			}
			if tt.endpoint != "" {
				cfg.Endpoint = &tt.endpoint
			}
			assert.Equal(t, tt.want, gcsBaseURL(cfg))
		})
	}
}

type fakeRoundTripper struct {
	validObjectNames []string
	wantPrefix       string
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	errorResp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(bytes.NewBufferString("error")),
	}
	if !strings.HasPrefix(req.URL.String(), f.wantPrefix) {
		return errorResp, nil
	}

	objectName := req.URL.Query().Get("name")
	if !slices.Contains(f.validObjectNames, objectName) {
		return errorResp, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("ok")),
	}, nil
}

func TestGCSUpload(t *testing.T) {
	relPaths := []string{
		"2024-10-10/1234/test.png",
		"/test.png",
		"cause-error",
	}
	baseURL := "https://storage.googleapis.com/upload/storage/v1/b/test-bucket/o?uploadType=media&name="
	client := &http.Client{
		Transport: &fakeRoundTripper{
			wantPrefix:       baseURL,
			validObjectNames: relPaths[:2],
		},
	}

	tests := []struct {
		name     string
		baseURL  string
		relPaths []string
		wantErr  bool
	}{
		{
			name:     "valid",
			baseURL:  baseURL,
			relPaths: relPaths[:2],
			wantErr:  false,
		},
		{
			name:     "error-paths",
			baseURL:  baseURL,
			relPaths: relPaths[2:],
			wantErr:  true,
		},
		{
			name:     "error-url",
			baseURL:  strings.Replace(baseURL, "storage", "storage-staging", 1),
			relPaths: relPaths[:2],
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &GCS{
				client:  client,
				baseURL: tt.baseURL,
			}
			for _, path := range tt.relPaths {
				t.Run(path, func(t *testing.T) {
					if tt.wantErr {
						assert.Error(t, storage.upload(context.Background(), nil, path))
					} else {
						assert.NoError(t, storage.upload(context.Background(), nil, path))
					}
				})
			}
		})
	}
}
