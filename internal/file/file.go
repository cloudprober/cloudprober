// Copyright 2020-2023 The Cloudprober Authors.
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

/*
Package file implements utilities to read files from various backends.
*/
package file

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	b          []byte
	lastReload time.Time
}

var global = struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
}{
	cache: make(map[string]cacheEntry),
}

type readFunc func(ctx context.Context, path string) ([]byte, error)
type modTimeFunc func(ctx context.Context, path string) (time.Time, error)

var zeroTime = time.Time{}

var prefixToReadfunc = map[string]readFunc{
	"gs://":    readFileFromGCS,
	"s3://":    readFileFromS3,
	"http://":  readFileFromHTTP,
	"https://": readFileFromHTTP,
}

var prefixToModTimeFunc = map[string]modTimeFunc{
	"gs://":    gcsModTime,
	"s3://":    s3ModTime,
	"http://":  httpModTime,
	"https://": httpModTime,
}

func parseObjectURL(objectPath string) (bucket, object string, err error) {
	parts := strings.SplitN(objectPath, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid object URL: %s", objectPath)
	}
	return parts[0], parts[1], nil
}

func substituteEnvVariables(content []byte) []byte {
	return []byte(os.ExpandEnv(string(content)))
}

type readFileOpts struct {
	substituteEnvVariables bool
}

type ReadOption func(*readFileOpts)

func WithEnvSubstitution() ReadOption {
	return func(opts *readFileOpts) {
		opts.substituteEnvVariables = true
	}
}

// ReadFile returns file contents as a slice of bytes. It's similar to ioutil's
// ReadFile, but includes support for files on non-disk locations. For example,
// files with paths starting with gs:// are assumed to be on GCS, and are read
// from GCS.
func ReadFile(ctx context.Context, fname string, readOptions ...ReadOption) ([]byte, error) {
	opts := &readFileOpts{}
	for _, ropt := range readOptions {
		ropt(opts)
	}

	processContent := func(b []byte, err error) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		if opts.substituteEnvVariables {
			b = []byte(os.ExpandEnv(string(b)))
		}
		return b, nil
	}

	for prefix, f := range prefixToReadfunc {
		if strings.HasPrefix(fname, prefix) {
			return processContent(f(ctx, fname))
		}
	}
	return processContent(os.ReadFile(fname))
}

func ReadWithCache(ctx context.Context, fname string, refreshInterval time.Duration, readOptions ...ReadOption) ([]byte, error) {
	global.mu.RLock()
	fc, ok := global.cache[fname]
	global.mu.RUnlock()
	if ok && (time.Since(fc.lastReload) < refreshInterval) {
		return fc.b, nil
	}

	b, err := ReadFile(ctx, fname, readOptions...)
	if err != nil {
		return nil, err
	}

	global.mu.Lock()
	global.cache[fname] = cacheEntry{b: b, lastReload: time.Now()}
	global.mu.Unlock()
	return b, nil
}

// ModTime returns file's modified timestamp.
func ModTime(ctx context.Context, fname string) (time.Time, error) {
	for prefix, f := range prefixToModTimeFunc {
		if strings.HasPrefix(fname, prefix) {
			return f(ctx, fname)
		}
	}

	statInfo, err := os.Stat(fname)
	if err != nil {
		return time.Time{}, err
	}
	return statInfo.ModTime(), nil
}
