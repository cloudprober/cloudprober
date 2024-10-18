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
	"net/http"
	"path"

	"github.com/cloudprober/cloudprober/internal/oauth"
	oauthconfigpb "github.com/cloudprober/cloudprober/internal/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"golang.org/x/oauth2"
)

type gcsStorage struct {
	client  *http.Client
	path    string
	baseURL string
}

func gcsBaseURL(cfg *configpb.GCS) string {
	return fmt.Sprintf("%s/upload/storage/v1/b/%s/o?uploadType=media&name=", cfg.GetEndpoint(), cfg.GetBucket())
}

func initGCS(ctx context.Context, cfg *configpb.GCS, l *logger.Logger) (*gcsStorage, error) {
	if cfg.GetBucket() == "" {
		return nil, fmt.Errorf("GCS bucket name is required")
	}

	if cfg.GetCredentials().GetScope() == nil {
		cfg.Credentials.Scope = []string{"https://www.googleapis.com/auth/devstorage.read_write"}
	}
	oauthCfg := oauthconfigpb.Config{
		Type: &oauthconfigpb.Config_GoogleCredentials{
			GoogleCredentials: cfg.Credentials,
		},
	}
	oauthTS, err := oauth.TokenSourceFromConfig(&oauthCfg, l)
	if err != nil {
		return nil, err
	}

	client := oauth2.NewClient(ctx, oauthTS)

	return &gcsStorage{
		client:  client,
		path:    cfg.GetPath(),
		baseURL: gcsBaseURL(cfg),
	}, nil
}

func (s *gcsStorage) upload(ctx context.Context, r io.Reader, relPath string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+path.Join(s.path, relPath), r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload object, status code: %d, msg: %s", resp.StatusCode, string(b))
	}
	return nil
}

// store syncs a local directory to an S3 path
func (s *gcsStorage) store(ctx context.Context, localPath, basePath string) error {
	return walkAndSave(ctx, localPath, basePath, func(ctx context.Context, r io.Reader, relPath string) error {
		return s.upload(ctx, r, relPath)
	})
}
