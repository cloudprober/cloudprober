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
	"bytes"
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
	l           *logger.Logger
}

const (
	identityEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
	storageURL       = "https://storage.azure.com/"
	tokenAPIVersion  = "2018-02-01"
	version          = "2020-08-04"
)

// stringToSign creates the string to sign for the given request.
// https://docs.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key#shared-key-format-for-authorization
func (s *absStorage) stringToSign(req *http.Request) string {
	cHeaders := fmt.Sprintf("x-ms-blob-type:BlockBlob\nx-ms-date:%s\nx-ms-version:%s\n", req.Header.Get("x-ms-date"), version)
	cResource := path.Join(s.accountName, req.URL.Path)
	return fmt.Sprintf("%s\n\n\n%d\n\n\n\n\n\n\n\n\n%s/%s", req.Method, req.ContentLength, cHeaders, cResource)
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
		l:           l,
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
						"resource=" + storageURL,
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

// signature creates signature for the given string using the account key.
func (s *absStorage) signature(stringToSign string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *absStorage) upload(ctx context.Context, r io.Reader, relPath string) error {
	fileContent, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file content: %v", err)
	}
	blobPath := path.Join(s.container, s.path, relPath)

	req, err := http.NewRequestWithContext(ctx, "PUT", s.endpoint+"/"+blobPath, bytes.NewReader(fileContent))
	if err != nil {
		return err
	}

	if req.ContentLength == 0 {
		req.ContentLength = int64(len(fileContent))
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", req.ContentLength))
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", version)
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", date)

	if s.key != nil {
		stringToSign := s.stringToSign(req)
		s.l.Debugf("String to sign: %s", stringToSign)
		req.Header.Set("Authorization", fmt.Sprintf("SharedKey %s:%s", s.accountName, s.signature(stringToSign)))
	}

	s.l.Infof("Sending request to: %s, with headers: %v", req.URL, req.Header)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
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
