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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudprober/cloudprober/logger"
)

// RemovePathSegmentFn returns a function that removes a segment from the path
// and returns the relative path.
// basePath is the base path to which the relative path is computed.
// segment is the segment to be removed from the path.
func RemovePathSegmentFn(basePath, segment string) func(string) (string, error) {
	return func(localPath string) (string, error) {
		sep := string(filepath.Separator) // / on Unix, \ on Windows
		return filepath.Rel(basePath, strings.Replace(localPath, sep+segment+sep, sep, 1))
	}
}

func walkAndSave(ctx context.Context, localPath string, destPathFn func(string) (string, error), fn func(context.Context, io.Reader, string) error) error {
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
		relPath, err := destPathFn(localFilePath)
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

type Local struct {
	destDir     string
	storagePath string
	l           *logger.Logger
}

func InitLocal(destDir, storagePath string, l *logger.Logger) (*Local, error) {
	s := &Local{
		destDir:     destDir,
		storagePath: storagePath,
		l:           l,
	}
	// Verify that the destination directory exists
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("destination directory %s does not exist", destDir)
	}

	// Create the storage directory if it does not exist
	// We create it here as cleanup handler starts right after initialization.
	if err := os.MkdirAll(filepath.Join(destDir, storagePath), 0755); err != nil {
		return nil, fmt.Errorf("error creating storage directory (%s): %v", filepath.Join(destDir, storagePath), err)
	}

	return s, nil
}

func (s *Local) saveFile(r io.Reader, relPath string) error {
	filePath := filepath.Join(s.destDir, s.storagePath, relPath)

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
func (s *Local) Store(ctx context.Context, localPath string, destPathFn func(string) (string, error)) error {
	s.l.Infof("Saving artifacts from %s at: %s", localPath, s.destDir)

	return walkAndSave(ctx, localPath, destPathFn, func(ctx context.Context, r io.Reader, relPath string) error {
		return s.saveFile(r, relPath)
	})
}
