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
	basePath  string
	s3Storage []*s3Storage
}

func initArtifactsHandler(opts *configpb.ArtifactsOptions, probeName, outputDir string, l *logger.Logger) (*artifactsHandler, error) {
	ah := &artifactsHandler{
		basePath: outputDir,
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
			s3, err := initS3(context.Background(), s3conf, l)

			if err != nil {
				return nil, err
			}

			ah.s3Storage = append(ah.s3Storage, s3)
		}
	}

	return ah, nil
}

func (ah *artifactsHandler) handle(path string) {
	for _, s3 := range ah.s3Storage {
		s3 := s3 // create a new variable to avoid capturing the loop variable
		go func(s3 *s3Storage) {
			if err := s3.uploadDirToS3(context.Background(), path, ah.basePath); err != nil {
				s3.l.Errorf("error uploading artifacts to S3: %v", err)
			}
		}(s3)
	}
}
