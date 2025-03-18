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

package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestNewCleanupHandler(t *testing.T) {
	testDir := "/tmp/x"
	tests := []struct {
		name    string
		opts    *configpb.CleanupOptions
		dir     string
		want    *CleanupHandler
		wantErr bool
	}{
		{
			name: "max_age_sec cannot be 0",
			opts: &configpb.CleanupOptions{
				MaxAgeSec:          proto.Int32(0),
				CleanupIntervalSec: proto.Int32(1),
			},
			wantErr: true,
		},
		{
			name: "cleanup_interval_sec cannot be 0",
			opts: &configpb.CleanupOptions{
				MaxAgeSec:          proto.Int32(1),
				CleanupIntervalSec: proto.Int32(0),
			},
			wantErr: true,
		},
		{
			name: "cleanup_interval_sec cannot be greater than max_age_sec",
			opts: &configpb.CleanupOptions{
				MaxAgeSec:          proto.Int32(1),
				CleanupIntervalSec: proto.Int32(2),
			},
			wantErr: true,
		},
		{
			name: "cleanup_interval_sec is nil",
			opts: &configpb.CleanupOptions{
				MaxAgeSec: proto.Int32(1),
			},
			dir: testDir,
			want: &CleanupHandler{
				dir:      testDir,
				interval: time.Second,
				maxAge:   time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCleanupHandler(tt.dir, tt.opts, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("newCleanupHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestCleanupHandlerCleanupCycle(t *testing.T) {
	dir, err := os.MkdirTemp("", "cleanup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	subDirs := []string{"dt1", "dt2", "dt3"}
	for i, subDir := range subDirs {
		path := filepath.Join(dir, subDir)
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		file, err := os.Create(filepath.Join(path, fmt.Sprintf("ft%d", i+1)))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		file.Close()
		time.Sleep(100 * time.Millisecond)
	}

	maxAge := 150 * time.Millisecond
	ch := &CleanupHandler{
		dir:    dir,
		maxAge: maxAge,
	}
	ch.cleanupCycle()

	remainingDirs, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	expectedDirs := map[string]bool{"dt3": true}
	for _, d := range remainingDirs {
		if !expectedDirs[d.Name()] {
			t.Errorf("unexpected directory: %s", d.Name())
		}
		delete(expectedDirs, d.Name())
	}

	for d := range expectedDirs {
		t.Errorf("expected directory not found: %s", d)
	}
}
