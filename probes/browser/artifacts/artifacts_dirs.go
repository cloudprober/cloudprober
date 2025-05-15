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

package artifacts

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type DirEntry struct {
	Path    string
	ModTime time.Time
}

func getTimestampDirectories(root string, startTime, endTime time.Time, max int) ([]DirEntry, error) {
	var timestampDirs []DirEntry

	// Get first level directories (dates)
	dateDirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, dateDir := range dateDirs {
		if !dateDir.IsDir() {
			continue
		}

		// Get date directory info for time checking
		dateInfo, err := dateDir.Info()
		if err != nil {
			return nil, err
		}

		dateModTime := dateInfo.ModTime()

		// Skip date directory if it's entirely outside the time range
		if !startTime.IsZero() && dateModTime.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && dateModTime.After(endTime) {
			continue
		}

		datePath := filepath.Join(root, dateDir.Name())

		// Get second level directories (timestamps)
		tsDirs, err := os.ReadDir(datePath)
		if err != nil {
			return nil, err
		}

		for _, tsDir := range tsDirs {
			if !tsDir.IsDir() {
				continue
			}

			// Check if directory name is a valid Unix timestamp
			if _, err := strconv.ParseInt(tsDir.Name(), 10, 64); err != nil {
				continue
			}

			info, err := tsDir.Info()
			if err != nil {
				return nil, err
			}

			modTime := info.ModTime()

			// Check time bounds (zero time means no bound)
			if !startTime.IsZero() && modTime.Before(startTime) {
				continue
			}
			if !endTime.IsZero() && modTime.After(endTime) {
				continue
			}

			fullPath := filepath.Join(datePath, tsDir.Name())
			timestampDirs = append(timestampDirs, DirEntry{
				Path:    fullPath,
				ModTime: modTime,
			})
		}
	}

	// Sort directories by modification time in reverse chronological order
	sort.Slice(timestampDirs, func(i, j int) bool {
		return timestampDirs[i].ModTime.After(timestampDirs[j].ModTime)
	})

	// Apply max limit
	if max > 0 && len(timestampDirs) > max {
		timestampDirs = timestampDirs[:max]
	}

	return timestampDirs, nil
}
