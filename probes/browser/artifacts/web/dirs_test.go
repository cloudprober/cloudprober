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
	"os"
	"path/filepath"
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
	root := t.TempDir()

	// Structure: map[date][]timestamps
	structure := map[string][]int64{
		"2025-05-14": {1111, 2222},
		"2025-05-15": {3333},
		"invalid":    {4444}, // Should be ignored
	}
	modTimes := map[string]time.Time{
		"2025-05-14/1111": time.Date(2025, 5, 14, 10, 0, 0, 0, time.UTC),
		"2025-05-14/2222": time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC),
		"2025-05-15/3333": time.Date(2025, 5, 15, 9, 0, 0, 0, time.UTC),
		"invalid/4444":    time.Date(2025, 5, 13, 8, 0, 0, 0, time.UTC),
	}
	createTestDirStructure(t, root, structure, modTimes)

	tests := []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		max       int
		expect    []string // Expected paths, sorted by modtime desc
	}{
		{
			name:      "all entries, no bounds",
			startTime: time.Time{},
			endTime:   time.Time{},
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-15", "3333"),
				filepath.Join(root, "2025-05-14", "2222"),
				filepath.Join(root, "2025-05-14", "1111"),
			},
		},
		{
			name:      "start bound",
			startTime: time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC),
			endTime:   time.Time{},
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-15", "3333"),
				filepath.Join(root, "2025-05-14", "2222"),
			},
		},
		{
			name:      "end bound",
			startTime: time.Time{},
			endTime:   time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC),
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-14", "2222"),
				filepath.Join(root, "2025-05-14", "1111"),
			},
		},
		{
			name:      "start and end bounds",
			startTime: time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC),
			endTime:   time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC),
			max:       0,
			expect: []string{
				filepath.Join(root, "2025-05-14", "2222"),
			},
		},
		{
			name:      "start and end bounds - no match",
			startTime: time.Date(2025, 5, 14, 11, 20, 0, 0, time.UTC),
			endTime:   time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC),
			max:       0,
			expect:    nil,
		},
		{
			name:      "start and end bounds - no match - startTime after endTime",
			startTime: time.Date(2025, 5, 14, 12, 0, 0, 0, time.UTC),
			endTime:   time.Date(2025, 5, 14, 11, 0, 0, 0, time.UTC),
			max:       0,
			expect:    nil,
		},
		{
			name:      "max limit",
			startTime: time.Time{},
			endTime:   time.Time{},
			max:       2,
			expect: []string{
				filepath.Join(root, "2025-05-15", "3333"),
				filepath.Join(root, "2025-05-14", "2222"),
			},
		},
		{
			name:      "no matches",
			startTime: time.Date(2025, 5, 16, 0, 0, 0, 0, time.UTC),
			endTime:   time.Time{},
			max:       0,
			expect:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dirs, err := getTimestampDirectories(root, tc.startTime, tc.endTime, tc.max)
			assert.NoError(t, err)
			var got []string
			for _, d := range dirs {
				got = append(got, d.Path)
			}
			assert.Equal(t, tc.expect, got)
		})
	}
}
