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

package browser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func walkAndSave(ctx context.Context, localPath, basePath string, fn func(context.Context, io.Reader, string) error) error {
	// Check if the local directory exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local directory %s does not exist", localPath)
	}

	// Walk the local directory and upload each file to S3
	err := filepath.Walk(localPath, func(localFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		// Get the relative path of the file
		relPath, err := filepath.Rel(basePath, localFilePath)
		if err != nil {
			return err
		}

		file, err := os.Open(localFilePath)
		if err != nil {
			return err
		}
		defer file.Close()

		return fn(ctx, file, relPath)
	})

	return err
}

type localStorage struct {
	destDir string
}

func initLocalStorage(destDir string) (*localStorage, error) {
	s := &localStorage{
		destDir: destDir,
	}
	// Verify that the destination directory exists
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("destination directory %s does not exist", destDir)
	}

	return s, nil
}

func (s *localStorage) saveFile(r io.Reader, relPath string) error {
	filePath := filepath.Join(s.destDir, relPath)

	// Create the destination directory and all necessary parent directories
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	fileBytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Write to file path
	if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
		return err
	}

	return nil
}

// store saves the local directory to the destination directory.
func (s *localStorage) store(ctx context.Context, localPath, basePath string) error {
	return walkAndSave(ctx, localPath, basePath, func(ctx context.Context, r io.Reader, relPath string) error {
		return s.saveFile(r, relPath)
	})
}
