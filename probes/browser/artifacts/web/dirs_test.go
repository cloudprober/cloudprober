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

package web

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTestDirStructure(t *testing.T, root string, structure map[string][]int64, modTimes map[string]time.Time) {
	for date, timestamps := range structure {
		dateDir := filepath.Join(root, date)
		if err := os.Mkdir(dateDir, 0o755); err != nil {
			t.Fatalf("Failed to create date dir: %v", err)
		}
		for _, ts := range timestamps {
			tsStr := strconv.FormatInt(ts, 10)
			tsDir := filepath.Join(dateDir, tsStr)
			if err := os.Mkdir(tsDir, 0o755); err != nil {
				t.Fatalf("Failed to create ts dir: %v", err)
			}
			if mt, ok := modTimes[filepath.Join(date, tsStr)]; ok {
				if err := os.Chtimes(tsDir, mt, mt); err != nil {
					t.Fatalf("Failed to set modtime: %v", err)
				}
			}
		}
	}
}

func TestGetTimestampDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows as changing modtime doesn't work reliably")
	}

	root := t.TempDir()

	now := time.Now()
	currDateDir := now.Format("2006-01-02")

	// Structure: map[date][]timestamps
	structure := map[string][]int64{
		"2025-05-14": {1111, 2222},
		"2025-05-15": {3333},
		currDateDir:  {4444},
		"invalid":    {5555}, // Should be ignored
	}
	modTimes := map[string]time.Time{
		"2025-05-14/1111":     time.Date(2025, 5, 14, 10, 0, 0, 0, time.UTC),
		"2025-05-14/2222":     time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC),
		"2025-05-15/3333":     time.Date(2025, 5, 15, 9, 0, 0, 0, time.UTC),
		currDateDir + "/4444": now,
		"invalid/5555":        time.Date(2025, 5, 13, 8, 0, 0, 0, time.UTC),
	}
	createTestDirStructure(t, root, structure, modTimes)

	tests := []struct {
		name      string
		startTime string
		endTime   string
		max       int
		expect    []string // Expected paths, sorted by modtime desc
	}{
		{
			name:      "all entries",
			startTime: "0",
			expect: []string{
				filepath.Join(root, currDateDir, "4444"),
				filepath.Join(root, "2025-05-15", "3333"),
				filepath.Join(root, "2025-05-14", "2222"),
				filepath.Join(root, "2025-05-14", "1111"),
			},
		},
		{
			name: "default bounds",
			expect: []string{
				filepath.Join(root, currDateDir, "4444"),
			},
		},
		{
			name:      "start bound",
			startTime: strconv.FormatInt(time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC).UnixMilli(), 10),
			expect: []string{
				filepath.Join(root, currDateDir, "4444"),
				filepath.Join(root, "2025-05-15", "3333"),
				filepath.Join(root, "2025-05-14", "2222"),
			},
		},
		{
			name:      "end bound",
			startTime: "0",
			endTime:   strconv.FormatInt(time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC).UnixMilli(), 10),
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-14", "2222"),
				filepath.Join(root, "2025-05-14", "1111"),
			},
		},
		{
			name:      "start and end bounds",
			startTime: strconv.FormatInt(time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC).UnixMilli(), 10),
			endTime:   strconv.FormatInt(time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC).UnixMilli(), 10),
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-14", "2222"),
			},
		},
		{
			name:      "start and end bounds - no match",
			startTime: strconv.FormatInt(time.Date(2025, 5, 14, 11, 20, 0, 0, time.UTC).UnixMilli(), 10),
			endTime:   strconv.FormatInt(time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC).UnixMilli(), 10),
			max:       0,
			expect:    nil,
		},
		{
			name:      "start and end bounds - no match - startTime after endTime",
			startTime: strconv.FormatInt(time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC).UnixMilli(), 10),
			endTime:   strconv.FormatInt(time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC).UnixMilli(), 10),
			max:       0,
			expect:    nil,
		},
		{
			name:      "max limit",
			startTime: "0",
			max:       2,
			expect: []string{
				filepath.Join(root, currDateDir, "4444"),
				filepath.Join(root, "2025-05-15", "3333"),
			},
		},
		{
			name:      "no matches",
			startTime: strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			expect:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{}
			if tc.startTime != "" {
				query.Set("startTime", tc.startTime)
			}
			if tc.endTime != "" {
				query.Set("endTime", tc.endTime)
			}
			dirs, err := getTimestampDirectories(root, query, tc.max)
			assert.NoError(t, err)
			var got []string
			for _, d := range dirs {
				got = append(got, d.Path)
			}
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestProbeFailed(t *testing.T) {
	root := t.TempDir()

	// Case 1: No failure marker
	dir1 := filepath.Join(root, "dir1")
	if err := os.Mkdir(dir1, 0o755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}
	assert.False(t, probeFailed(dir1), "probeFailed should be false when no marker exists")

	// Case 2: Failure marker in root
	marker := filepath.Join(dir1, FailureMarkerFile)
	if f, err := os.Create(marker); err != nil {
		t.Fatalf("Failed to create failure marker: %v", err)
	} else {
		f.Close()
	}
	assert.True(t, probeFailed(dir1), "probeFailed should be true when marker exists in root")

	// Case 3: Failure marker in subdirectory
	dir2 := filepath.Join(root, "dir2")
	subdir := filepath.Join(dir2, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	assert.False(t, probeFailed(dir2), "probeFailed should be false when no marker exists anywhere")
	marker2 := filepath.Join(subdir, FailureMarkerFile)
	if f, err := os.Create(marker2); err != nil {
		t.Fatalf("Failed to create failure marker in subdir: %v", err)
	} else {
		f.Close()
	}
	assert.True(t, probeFailed(dir2), "probeFailed should be true when marker exists in subdir")

	// Case 4: Directory does not exist
	nonexistent := filepath.Join(root, "doesnotexist")
	assert.False(t, probeFailed(nonexistent), "probeFailed should be false for nonexistent directory")
}
