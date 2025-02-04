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

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/logger"
)

type artifactsHandler struct {
	basePath     string
	s3Storage    []*s3Storage
	gcsStorage   []*gcsStorage
	absStorage   []*absStorage
	localStorage []*localStorage
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
			s3, err := initS3(context.Background(), s3conf)

			if err != nil {
				return fmt.Errorf("error initializing S3 storage (bucket: %s): %v", s3conf.GetBucket(), err)
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}

		if gcsConf := storageConfig.GetGcs(); gcsConf != nil {
			gcs, err := initGCS(context.Background(), gcsConf, p.l)
			if err != nil {
				return fmt.Errorf("error initializing GCS storage: %v", err)
			}
			ah.gcsStorage = append(ah.gcsStorage, gcs)
		}

		if absConf := storageConfig.GetAbs(); absConf != nil {
			abs, err := initABS(context.Background(), absConf, p.l)
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

			ls, err := initLocalStorage(localStorage.GetDir())
			if err != nil {
				return fmt.Errorf("error initializing local storage: %v", err)
			}
			localStorageDirs = append(localStorageDirs, ls.destDir)
			ah.localStorage = append(ah.localStorage, ls)
		}
	}

	if p.c.GetArtifactsOptions().GetServeOnWeb() {
		httpServerMux := runconfig.DefaultHTTPServeMux()
		if httpServerMux == nil {
			return fmt.Errorf("default http server mux is not initialized")
		}

		webRoot, err := p.webServerRoot(localStorageDirs)
		if err != nil {
			return err
		}
		fileServer := http.FileServer(http.Dir(webRoot))
		pathPrefix := p.pathPrefix()
		httpServerMux.Handle(pathPrefix+"/", http.StripPrefix(pathPrefix, fileServer))
	}

	p.artifactsHandler = ah
	return nil
}

func (ah *artifactsHandler) handle(ctx context.Context, path string) {
	for _, s3 := range ah.s3Storage {
		go func(s3 *s3Storage) {
			ah.l.Infof("Uploading artifacts from %s to: s3://%s/%s", path, s3.bucket, s3.path)
			if err := s3.store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to S3: %v", err)
			}
		}(s3)
	}

	for _, gcs := range ah.gcsStorage {
		go func(gcs *gcsStorage) {
			ah.l.Infof("Uploading artifacts from %s to: %s", path, gcs.baseURL)
			if err := gcs.store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to GCS: %v", err)
			}
		}(gcs)
	}

	for _, abs := range ah.absStorage {
		go func(abs *absStorage) {
			ah.l.Infof("Uploading artifacts from %s to: %s", path, abs.endpoint)
			if err := abs.store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error uploading artifacts to ABS: %v", err)
			}
		}(abs)
	}

	for _, lStorage := range ah.localStorage {
		go func(lStorage *localStorage) {
			ah.l.Infof("Saving artifacts from %s at: %s", path, lStorage.destDir)
			if err := lStorage.store(ctx, path, ah.basePath); err != nil {
				ah.l.Errorf("error saving artifacts locally: %v", err)
			}
		}(lStorage)
	}
}
