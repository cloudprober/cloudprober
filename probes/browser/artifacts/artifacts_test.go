// Copyright 2024-2025 The Cloudprober Authors.
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

package artifacts

import (
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestPathPrefix(t *testing.T) {
	probeName := "test_probe"

	tests := []struct {
		name           string
		artifactsOpts  *configpb.ArtifactsOptions
		expectedPrefix string
	}{
		{
			name:           "default path prefix",
			artifactsOpts:  &configpb.ArtifactsOptions{},
			expectedPrefix: "/artifacts/" + probeName,
		},
		{
			name: "custom path prefix",
			artifactsOpts: &configpb.ArtifactsOptions{
				WebServerPath: proto.String("custom/path"),
			},
			expectedPrefix: "custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedPrefix, pathPrefix(tt.artifactsOpts, probeName))
		})
	}
}

func TestWebServerRoot(t *testing.T) {
	outputDir := "/output/dir"

	tests := []struct {
		name             string
		artifactsOpts    *configpb.ArtifactsOptions
		outputDir        string
		localStorageDirs []string
		expectedRoot     string
		expectError      bool
	}{
		{
			name:          "no local storage",
			artifactsOpts: &configpb.ArtifactsOptions{},
			outputDir:     outputDir,
			expectedRoot:  outputDir,
			expectError:   false,
		},
		{
			name:          "no local storage, no default",
			artifactsOpts: &configpb.ArtifactsOptions{},
			outputDir:     "",
			expectedRoot:  "",
			expectError:   true,
		},
		{
			name:             "only one local storage",
			artifactsOpts:    &configpb.ArtifactsOptions{},
			localStorageDirs: []string{"/local/storage/dir"},
			outputDir:        outputDir,
			expectedRoot:     "/local/storage/dir",
			expectError:      false,
		},
		{
			name: "custom valid web server root",
			artifactsOpts: &configpb.ArtifactsOptions{
				WebServerRoot: proto.String("/local/storage/dir"),
			},
			outputDir:        outputDir,
			localStorageDirs: []string{"/local/storage/dir"},
			expectedRoot:     "/local/storage/dir",
			expectError:      false,
		},
		{
			name: "custom invalid web server root",
			artifactsOpts: &configpb.ArtifactsOptions{
				WebServerRoot: proto.String("/invalid/storage/dir"),
			},
			localStorageDirs: []string{"/local/storage/dir"},
			outputDir:        outputDir,
			expectedRoot:     "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, lsDir := range tt.localStorageDirs {
				tt.artifactsOpts.Storage = append(tt.artifactsOpts.Storage, &configpb.Storage{
					Storage: &configpb.Storage_LocalStorage{
						LocalStorage: &configpb.LocalStorage{
							Dir: proto.String(lsDir),
						},
					},
				})
			}
			root, err := webServerRoot(tt.artifactsOpts, tt.outputDir)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoot, root)
			}
		})
	}
}
