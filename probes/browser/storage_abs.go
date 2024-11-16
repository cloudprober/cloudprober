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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/cloudprober/cloudprober/internal/oauth"
	oauthconfigpb "github.com/cloudprober/cloudprober/internal/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"golang.org/x/oauth2"
)

type absStorage struct {
	container   string
	accountName string
	client      *http.Client
	key         []byte
	path        string
	endpoint    string
}

const (
	identityEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
	resource         = "https://storage.azure.com/"
	tokenAPIVersion  = "2018-02-01"
	contentType      = "application/octet-stream"
	version          = "2020-08-04"
)

func (s *absStorage) stringToSign(contentLength int64, contentType, canonicalizedHeaders, blobPath string) string {
	return fmt.Sprintf("PUT\n\n\n%d\n\n%s\n\n\n\n\n\n\n%s/%s", contentLength, contentType, canonicalizedHeaders, path.Join(s.accountName, blobPath))
}

func initABS(ctx context.Context, cfg *configpb.ABS, l *logger.Logger) (*absStorage, error) {
	if cfg.GetContainer() == "" {
		return nil, fmt.Errorf("ABS container name is required")
	}

	abs := &absStorage{
		container:   cfg.GetContainer(),
		accountName: cfg.GetAccountName(),
		path:        cfg.GetPath(),
		endpoint:    cfg.GetEndpoint(),
		client:      &http.Client{},
	}
	if abs.endpoint == "" {
		abs.endpoint = "https://" + abs.accountName + ".blob.core.windows.net"
	}

	// If account key is not provided, we will use the metadata service to get
	// the bearer token.
	if cfg.GetAccountKey() == "" {
		oauthTS, err := oauth.TokenSourceFromConfig(&oauthconfigpb.Config{
			Type: &oauthconfigpb.Config_HttpRequest{
				HttpRequest: &oauthconfigpb.HTTPRequest{
					TokenUrl: identityEndpoint,
					Header: map[string]string{
						"Metadata": "true",
					},
					Data: []string{
						"resource=" + resource,
						"api-version=" + tokenAPIVersion,
					},
				},
			},
		}, l)
		if err != nil {
			return nil, err
		}
		abs.client = oauth2.NewClient(ctx, oauthTS)
	} else {
		key, err := base64.StdEncoding.DecodeString(cfg.GetAccountKey())
		if err != nil {
			return nil, fmt.Errorf("failed to decode account key: %v", err)
		}
		abs.key = key
	}

	return abs, nil
}

// CreateAuthorizationHeader generates the authorization header needed for Azure Blob REST API requests.
func (s *absStorage) createAuthorizationHeader(req *http.Request) string {
	canonicalizedHeaders := ""
	for header, values := range req.Header {
		if len(values) > 0 && header[:5] == "x-ms-" {
			canonicalizedHeaders += fmt.Sprintf("%s:%s\n", header, values[0])
		}
	}

	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(s.stringToSign(req.ContentLength, contentType, canonicalizedHeaders, req.URL.Path)))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	authHeader := fmt.Sprintf("SharedKey %s:%s", s.accountName, signature)
	return authHeader
}

func (s *absStorage) upload(ctx context.Context, r io.Reader, relPath string) error {
	blobPath := path.Join(s.container, s.path, relPath)

	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint+"/"+blobPath, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", version)
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", date)

	if s.key != nil {
		req.Header.Set("Authorization", s.createAuthorizationHeader(req))
	}

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
func (s *absStorage) store(ctx context.Context, localPath, basePath string) error {
	return walkAndSave(ctx, localPath, basePath, func(ctx context.Context, r io.Reader, relPath string) error {
		return s.upload(ctx, r, relPath)
	})
}
