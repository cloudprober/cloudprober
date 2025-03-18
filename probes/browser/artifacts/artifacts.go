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
	"path/filepath"

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
)

type artifactsHandler struct {
	basePath     string
	s3Storage    []*s3Storage
	localStorage []*localStorage
	l            *logger.Logger
}

func initArtifactsHandler(opts *configpb.ArtifactsOptions, probeName, outputDir string, l *logger.Logger) (*artifactsHandler, error) {
	ah := &artifactsHandler{
		basePath: outputDir,
		l:        l,
	}

	if opts.GetServeOnWeb() {
		httpServerMux := runconfig.DefaultHTTPServeMux()
		if httpServerMux == nil {
			return nil, fmt.Errorf("default http server mux is not initialized")
		}

		pathPrefix := opts.GetWebServerPath()
		if pathPrefix == "" {
			pathPrefix = filepath.Join("/artifacts", probeName)
		}

		fileServer := http.FileServer(http.Dir(outputDir))
		httpServerMux.Handle(pathPrefix+"/", http.StripPrefix(pathPrefix, fileServer))
	}

	for _, storageConfig := range opts.GetStorage() {
		if s3conf := storageConfig.GetS3(); s3conf != nil {
			s3, err := initS3(context.Background(), s3conf)

			if err != nil {
				return nil, fmt.Errorf("error initializing S3 storage (bucket: %s): %v", s3conf.GetBucket(), err)
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}

		if dir := storageConfig.GetLocalStorageDir(); dir != "" {
			ls, err := initLocalStorage(dir)
			if err != nil {
				return nil, fmt.Errorf("error initializing local storage: %v", err)
			}
			ah.localStorage = append(ah.localStorage, ls)
		}
	}

	return ah, nil
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

	for _, lStorage := range ah.localStorage {
		go func(lStorage *localStorage) {
			ah.l.Infof("Saving artifacts from %s at: %s", path, lStorage.destDir)
			if err := lStorage.store(ctx, path, ah.basePath); err != nil {
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
