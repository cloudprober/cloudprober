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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkAndSave(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test directory structure
	testFiles := map[string]string{
		"date1/ts1/report/index.html":           "report1",
		"date1/ts1/report/data/file.png":        "image1",
		"date1/ts1/result/test1/screenshot.png": "image2",
		"date1/ts2/report/file3.txt":            "content1",
		"date2/ts1/result/file1.txt":            "content2",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	tests := []struct {
		name      string
		localPath string
		wantFiles map[string]string
		wantErr   bool
	}{
		{
			name:      "Valid walk and save - 1",
			localPath: filepath.Join(tempDir, "date1/ts1/report"),
			wantFiles: map[string]string{
				"date1/ts1/report/data/file.png": "image1",
				"date1/ts1/report/index.html":    "report1",
			},
		},
		{
			name:      "Valid walk and save - 2",
			localPath: filepath.Join(tempDir, "date1/ts2/report"),
			wantFiles: map[string]string{
				"date1/ts2/report/file3.txt": "content1",
			},
		},
		{
			name:      "Invalid walk and save",
			localPath: filepath.Join(tempDir, "date2/ts1/report"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filesSeen := make(map[string]string)
			fn := func(_ context.Context, r io.Reader, relPath string) error {
				content, err := io.ReadAll(r)
				if err != nil {
					return err
				}

				filesSeen[relPath] = string(content)
				return nil
			}

			err := walkAndSave(context.TODO(), tt.localPath, tempDir, fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("walkAndSave() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			wantFiles := make(map[string]string)
			for path, content := range tt.wantFiles {
				wantFiles[filepath.FromSlash(path)] = content
			}
			assert.Equal(t, wantFiles, filesSeen, "files seen")
		})
	}
}

func TestLocalStorageSaveFile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "date"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	localStorage, err := InitLocal(tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to initialize local storage: %v", err)
	}

	if err := localStorage.saveFile(bytes.NewReader([]byte("storage")), "date/ts1/report/file.txt"); err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// Verify the file content
	content, err := os.ReadFile(filepath.Join(tmpDir, "date", "ts1", "report", "file.txt"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "storage" {
		t.Errorf("File content mismatch. Expected 'storage', got '%s'", string(content))
	}
}
