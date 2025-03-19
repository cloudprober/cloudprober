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
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
)

type CleanupHandler struct {
	dir      string
	maxAge   time.Duration
	interval time.Duration
	l        *logger.Logger
}

func NewCleanupHandler(dir string, opts *configpb.CleanupOptions, l *logger.Logger) (*CleanupHandler, error) {
	if opts.GetMaxAgeSec() == 0 {
		return nil, errors.New("max_age_sec cannot be 0")
	}
	if opts.GetCleanupIntervalSec() == 0 {
		return nil, errors.New("cleanup_interval_sec cannot be 0")
	}

	ch := &CleanupHandler{
		dir:      dir,
		interval: time.Duration(opts.GetCleanupIntervalSec()) * time.Second,
		maxAge:   time.Duration(opts.GetMaxAgeSec()) * time.Second,
		l:        l,
	}

	if ch.maxAge < ch.interval {
		if opts.CleanupIntervalSec != nil {
			return nil, errors.New("cleanup_interval_sec cannot be greater than max_age_sec")
		}
		ch.interval = ch.maxAge
	}

	return ch, nil
}

func (ch *CleanupHandler) cleanupCycle() {
	oldestTime := time.Now().Add(-ch.maxAge)
	filepath.Walk(ch.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// We ignore errors as we may delete parent directory before
			// deleting its children, which will cause errors.
			return nil
		}
		if path == ch.dir {
			return nil
		}
		if info.ModTime().Before(oldestTime) {
			if info.IsDir() {
				if err := os.RemoveAll(path); err != nil {
					ch.l.Warningf("cleanupHandler: error cleaning up directory %s: %v", path, err)
				}
				return nil
			}
			if err := os.Remove(path); err != nil {
				ch.l.Warningf("cleanupHandler: error cleaning up file %s: %v", path, err)
			}
			return nil
		}
		return nil
	})
}

func (ch *CleanupHandler) Start(ctx context.Context) {
	ticker := time.NewTicker(ch.interval)
	for ; true; <-ticker.C {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		default:
		}

		ch.l.Infof("cleanupHandler: starting cleanup for %s", ch.dir)
		ch.cleanupCycle()
	}
}
