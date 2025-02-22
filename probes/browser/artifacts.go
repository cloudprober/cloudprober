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
	"net/http"
	"slices"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/browser/storage"
	"github.com/cloudprober/cloudprober/state"
)

type artifactsHandler struct {
	basePath     string
	s3Storage    []*storage.S3
	gcsStorage   []*storage.GCS
	absStorage   []*storage.ABS
	localStorage []*storage.Local
	l            *logger.Logger
}

func (p *Probe) pathPrefix() string {
	if pathPrefix := p.c.GetArtifactsOptions().GetWebServerPath(); pathPrefix != "" {
		return pathPrefix
	}
	return "/artifacts/" + p.name
}

func (p *Probe) webServerRoot(localStorageDirs []string) (string, error) {
	if r := p.c.GetArtifactsOptions().GetWebServerRoot(); r != "" {
		if !slices.Contains(localStorageDirs, r) {
			return "", fmt.Errorf("invalid web server root: %s; web server root can be either local_storage.dir or empty", r)
		}
		return r, nil
	}
	return p.outputDir, nil
}

func (p *Probe) initArtifactsHandler() error {
	ah := &artifactsHandler{
		basePath: p.outputDir,
		l:        p.l,
	}

	var localStorageDirs []string

	for _, storageConfig := range p.c.GetArtifactsOptions().GetStorage() {
		if s3conf := storageConfig.GetS3(); s3conf != nil {
			s3, err := storage.InitS3(context.Background(), s3conf, p.l)

			if err != nil {
				return fmt.Errorf("error initializing S3 storage (bucket: %s): %v", s3conf.GetBucket(), err)
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}

		if gcsConf := storageConfig.GetGcs(); gcsConf != nil {
			gcs, err := storage.InitGCS(context.Background(), gcsConf, p.l)
			if err != nil {
				return fmt.Errorf("error initializing GCS storage: %v", err)
			}
			ah.gcsStorage = append(ah.gcsStorage, gcs)
		}

		if absConf := storageConfig.GetAbs(); absConf != nil {
			abs, err := storage.InitABS(context.Background(), absConf, p.l)
			if err != nil {
				return fmt.Errorf("error initializing ABS storage: %v", err)
			}
			ah.absStorage = append(ah.absStorage, abs)
		}

		if localStorage := storageConfig.GetLocalStorage(); localStorage != nil {
			if localStorage.GetCleanupOptions() != nil {
				cleanupHandler, err := newCleanupHandler(localStorage.GetDir(), localStorage.GetCleanupOptions(), p.l)
				if err != nil {
					return fmt.Errorf("error initializing cleanup handler for local storage: %v", err)
				}
				p.cleanupHandlers = append(p.cleanupHandlers, cleanupHandler)
			}

			ls, err := storage.InitLocal(localStorage.GetDir(), p.l)
			if err != nil {
				return fmt.Errorf("error initializing local storage: %v", err)
			}
			localStorageDirs = append(localStorageDirs, localStorage.GetDir())
			ah.localStorage = append(ah.localStorage, ls)
		}
	}

	if p.c.GetArtifactsOptions().GetServeOnWeb() {
		webRoot, err := p.webServerRoot(localStorageDirs)
		if err != nil {
			return err
		}
		fileServer := http.FileServer(http.Dir(webRoot))
		pathPrefix := p.pathPrefix()
		if err := state.AddWebHandler(pathPrefix+"/", http.StripPrefix(pathPrefix, fileServer).ServeHTTP); err != nil {
			return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
		}
	}

	p.artifactsHandler = ah
	return nil
}

func (ah *artifactsHandler) handle(ctx context.Context, path string) {
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
