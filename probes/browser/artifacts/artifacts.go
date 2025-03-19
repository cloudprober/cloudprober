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
	"net/http"
	"slices"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"github.com/cloudprober/cloudprober/probes/browser/artifacts/storage"
	"github.com/cloudprober/cloudprober/state"
)

type ArtifactsHandler struct {
	basePath        string
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

func webServerRoot(opts *configpb.ArtifactsOptions, localStorageDirs []string, outputDir string) (string, error) {
	if r := opts.GetWebServerRoot(); r != "" {
		if !slices.Contains(localStorageDirs, r) {
			return "", fmt.Errorf("invalid web server root: %s; web server root can be either local_storage.dir or empty", r)
		}
		return r, nil
	}
	return outputDir, nil
}

func InitArtifactsHandler(opts *configpb.ArtifactsOptions, outputDir, probeName string, l *logger.Logger) (*ArtifactsHandler, error) {
	ah := &ArtifactsHandler{
		basePath: outputDir,
		l:        l,
	}

	var localStorageDirs []string

	for _, storageConfig := range opts.GetStorage() {
		if s3conf := storageConfig.GetS3(); s3conf != nil {
			s3, err := storage.InitS3(context.Background(), s3conf, l)

			if err != nil {
				return nil, fmt.Errorf("error initializing S3 storage (bucket: %s): %v", s3conf.GetBucket(), err)
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}

		if gcsConf := storageConfig.GetGcs(); gcsConf != nil {
			gcs, err := storage.InitGCS(context.Background(), gcsConf, l)
			if err != nil {
				return nil, fmt.Errorf("error initializing GCS storage: %v", err)
			}
			ah.gcsStorage = append(ah.gcsStorage, gcs)
		}

		if absConf := storageConfig.GetAbs(); absConf != nil {
			abs, err := storage.InitABS(context.Background(), absConf, l)
			if err != nil {
				return nil, fmt.Errorf("error initializing ABS storage: %v", err)
			}
			ah.absStorage = append(ah.absStorage, abs)
		}

		if localStorage := storageConfig.GetLocalStorage(); localStorage != nil {
			if localStorage.GetCleanupOptions() != nil {
				cleanupHandler, err := NewCleanupHandler(localStorage.GetDir(), localStorage.GetCleanupOptions(), l)
				if err != nil {
					return nil, fmt.Errorf("error initializing cleanup handler for local storage: %v", err)
				}
				ah.cleanupHandlers = append(ah.cleanupHandlers, cleanupHandler)
			}

			ls, err := storage.InitLocal(localStorage.GetDir(), l)
			if err != nil {
				return nil, fmt.Errorf("error initializing local storage: %v", err)
			}
			localStorageDirs = append(localStorageDirs, localStorage.GetDir())
			ah.localStorage = append(ah.localStorage, ls)
		}
	}

	if opts.GetServeOnWeb() {
		webRoot, err := webServerRoot(opts, localStorageDirs, outputDir)
		if err != nil {
			return nil, err
		}
		fileServer := http.FileServer(http.Dir(webRoot))
		pathPrefix := pathPrefix(opts, probeName)
		if err := state.AddWebHandler(pathPrefix+"/", http.StripPrefix(pathPrefix, fileServer).ServeHTTP); err != nil {
			return nil, fmt.Errorf("error adding web handler for artifacts web server: %v", err)
		}
	}

	return ah, nil
}

func (ah *ArtifactsHandler) Handle(ctx context.Context, path string) {
	for _, s3 := range ah.s3Storage {
		go func(s3 *storage.S3) {
			if err := s3.Store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to S3: %v", err)
			}
		}(s3)
	}

	for _, gcs := range ah.gcsStorage {
		go func(gcs *storage.GCS) {
			if err := gcs.Store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to GCS: %v", err)
			}
		}(gcs)
	}

	for _, abs := range ah.absStorage {
		go func(abs *storage.ABS) {
			if err := abs.Store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to ABS: %v", err)
			}
		}(abs)
	}

	for _, lStorage := range ah.localStorage {
		go func(lStorage *storage.Local) {
			if err := lStorage.Store(ctx, path, ah.basePath); err != nil {
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
