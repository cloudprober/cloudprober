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

package browser

import (
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestPathPrefix(t *testing.T) {
	tests := []struct {
		name           string
		artifactsOpts  *configpb.ArtifactsOptions
		expectedPrefix string
	}{
		{
			name:           "default path prefix",
			artifactsOpts:  &configpb.ArtifactsOptions{},
			expectedPrefix: "/artifacts/test_probe",
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
			p := &Probe{
				name: "test_probe",
				c: &configpb.ProbeConf{
					ArtifactsOptions: tt.artifactsOpts,
				},
			}
			assert.Equal(t, tt.expectedPrefix, p.pathPrefix())
		})
	}
}

func TestWebServerRoot(t *testing.T) {
	outputDir := "/output/dir"

	tests := []struct {
		name             string
		artifactsOpts    *configpb.ArtifactsOptions
		localStorageDirs []string
		expectedRoot     string
		expectError      bool
	}{
		{
			name:             "default web server root",
			artifactsOpts:    &configpb.ArtifactsOptions{},
			localStorageDirs: []string{"/local/storage/dir"},
			expectedRoot:     outputDir,
			expectError:      false,
		},
		{
			name: "custom valid web server root",
			artifactsOpts: &configpb.ArtifactsOptions{
				WebServerRoot: proto.String("/local/storage/dir"),
			},
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
			expectedRoot:     "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				outputDir: "/output/dir",
				c: &configpb.ProbeConf{
					ArtifactsOptions: tt.artifactsOpts,
				},
			}
			root, err := p.webServerRoot(tt.localStorageDirs)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoot, root)
			}
		})
	}
}
