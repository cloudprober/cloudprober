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
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"github.com/cloudprober/cloudprober/probes/browser/artifacts/storage"
	"github.com/cloudprober/cloudprober/probes/options"
	"google.golang.org/protobuf/proto"
)

var initGlobalServingOnce sync.Once

type ArtifactsHandler struct {
	destPathFn func(string) string

	s3Storage       []*storage.S3
	gcsStorage      []*storage.GCS
	absStorage      []*storage.ABS
	localStorage    []*storage.Local
	cleanupHandlers []*CleanupHandler
	l               *logger.Logger
}

func pathPrefix(opts *configpb.ArtifactsOptions, probeName string) string {
	if pathPrefix := opts.GetWebServerPath(); pathPrefix != "" {
		return pathPrefix
	}
	return "/artifacts/" + probeName
}

func webServerRoot(opts *configpb.ArtifactsOptions) (string, error) {
	var lsDirs []string
	for _, storage := range opts.GetStorage() {
		if localStorage := storage.GetLocalStorage(); localStorage != nil {
			lsDirs = append(lsDirs, filepath.Join(localStorage.GetDir(), storage.GetPath()))
		}
	}

	// User specified web server root
	if r := opts.GetWebServerRoot(); r != "" {
		if !slices.Contains(lsDirs, filepath.Clean(r)) {
			return "", fmt.Errorf("invalid web server root: %s; web server root can be either local_storage.dir or empty", r)
		}
		return r, nil
	}

	if len(lsDirs) >= 1 {
		return lsDirs[0], nil
	}

	return "", fmt.Errorf("no local storage directories configured, cannot determine web server root")
}

func globalToLocalOptions(in *configpb.ArtifactsOptions, pOpts *options.Options) *configpb.ArtifactsOptions {
	out := proto.Clone(in).(*configpb.ArtifactsOptions)
	for _, storage := range out.GetStorage() {
		storage.Path = proto.String(filepath.Join(storage.GetPath(), pOpts.Name))
	}
	// We serve artifacts at the global level, so disable them at probe level
	out.ServeOnWeb = proto.Bool(false)

	return out
}

// initGlobalArtifactsServing initializes global artifacts serving. It is idempotent.
func initGlobalArtifactsServing(opts *configpb.ArtifactsOptions, l *logger.Logger) error {
	if !opts.GetServeOnWeb() {
		return nil
	}

	var err error
	initGlobalServingOnce.Do(func() {
		var webRoot string
		webRoot, err = webServerRoot(opts)
		if err != nil {
			l.Errorf("error getting web server root: %v", err)
			return
		}
		err = serveArtifacts(pathPrefix(opts, ""), webRoot, true)
	})

	return err
}

func InitArtifactsHandler(ctx context.Context, opts *configpb.ArtifactsOptions, outputDir string, pOpts *options.Options, destPathFn func(string) string, l *logger.Logger) (*ArtifactsHandler, error) {
	ah := &ArtifactsHandler{
		destPathFn: destPathFn,
		l:          l,
	}

	// Following variable will be false, if opts is nil (no local artifacts
	// options) or if field is actually set to false
	probeLevelServeOnWeb := opts.GetServeOnWeb()

	if opts == nil {
		gopts := pOpts.ProberConfig.GetGlobalArtifactsOptions()
		if gopts == nil {
			return ah, nil
		}
		if err := initGlobalArtifactsServing(gopts, l); err != nil {
			return nil, err
		}
		opts = globalToLocalOptions(gopts, pOpts)
	}

	for _, storageConfig := range opts.GetStorage() {
		storagePath := storageConfig.GetPath()

		if s3conf := storageConfig.GetS3(); s3conf != nil {
			s3, err := storage.InitS3(ctx, s3conf, storagePath, l)

			if err != nil {
				return nil, fmt.Errorf("error initializing S3 storage (bucket: %s): %v", s3conf.GetBucket(), err)
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}

		if gcsConf := storageConfig.GetGcs(); gcsConf != nil {
			gcs, err := storage.InitGCS(ctx, gcsConf, storagePath, l)
			if err != nil {
				return nil, fmt.Errorf("error initializing GCS storage: %v", err)
			}
			ah.gcsStorage = append(ah.gcsStorage, gcs)
		}

		if absConf := storageConfig.GetAbs(); absConf != nil {
			abs, err := storage.InitABS(ctx, absConf, storagePath, l)
			if err != nil {
				return nil, fmt.Errorf("error initializing ABS storage: %v", err)
			}
			ah.absStorage = append(ah.absStorage, abs)
		}

		if localStorage := storageConfig.GetLocalStorage(); localStorage != nil {
			ls, err := storage.InitLocal(localStorage.GetDir(), storagePath, l)
			if err != nil {
				return nil, fmt.Errorf("error initializing local storage: %v", err)
			}

			if localStorage.GetCleanupOptions() != nil {
				dirToClean := filepath.Join(localStorage.GetDir(), storagePath)
				cleanupHandler, err := NewCleanupHandler(dirToClean, localStorage.GetCleanupOptions(), l)
				if err != nil {
					return nil, fmt.Errorf("error initializing cleanup handler for local storage: %v", err)
				}
				ah.cleanupHandlers = append(ah.cleanupHandlers, cleanupHandler)
			}

			ah.localStorage = append(ah.localStorage, ls)
		}
	}

	// probeLevelServeOnWeb will be true only if we're working with local
	// artifacts options and serve_on_web is set to true.
	if probeLevelServeOnWeb {
		webRoot, err := webServerRoot(opts)
		if err != nil {
			return nil, fmt.Errorf("error getting web server root: %v", err)
		}
		if err := serveArtifacts(pathPrefix(opts, pOpts.Name), webRoot, false); err != nil {
			return nil, err
		}
	}

	return ah, nil
}

func (ah *ArtifactsHandler) Handle(ctx context.Context, path string) {
	for _, s3 := range ah.s3Storage {
		go func(s3 *storage.S3) {
			if err := s3.Store(ctx, path, ah.destPathFn); err != nil {
				ah.l.Errorf("error uploading artifacts to S3: %v", err)
			}
		}(s3)
	}

	for _, gcs := range ah.gcsStorage {
		go func(gcs *storage.GCS) {
			if err := gcs.Store(ctx, path, ah.destPathFn); err != nil {
				ah.l.Errorf("error uploading artifacts to GCS: %v", err)
			}
		}(gcs)
	}

	for _, abs := range ah.absStorage {
		go func(abs *storage.ABS) {
			if err := abs.Store(ctx, path, ah.destPathFn); err != nil {
				ah.l.Errorf("error uploading artifacts to ABS: %v", err)
			}
		}(abs)
	}

	for _, lStorage := range ah.localStorage {
		go func(lStorage *storage.Local) {
			if err := lStorage.Store(ctx, path, ah.destPathFn); err != nil {
				ah.l.Errorf("error saving artifacts locally: %v", err)
			}
		}(lStorage)
	}
}

func (ah *ArtifactsHandler) StartCleanup(ctx context.Context) {
	for _, ch := range ah.cleanupHandlers {
		go ch.Start(ctx)
	}
}
