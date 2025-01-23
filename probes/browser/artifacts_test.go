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
