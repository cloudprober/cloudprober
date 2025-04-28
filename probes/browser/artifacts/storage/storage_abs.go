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

package storage

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

	"github.com/cloudprober/cloudprober/common/oauth"
	oauthconfigpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

type ABS struct {
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
func (s *ABS) stringToSign(req *http.Request) string {
	cHeaders := fmt.Sprintf("x-ms-blob-type:BlockBlob\nx-ms-date:%s\nx-ms-version:%s\n", req.Header.Get("x-ms-date"), version)
	cResource := path.Join(s.accountName, req.URL.Path)
	return fmt.Sprintf("%s\n\n\n%d\n\n\n\n\n\n\n\n\n%s/%s", req.Method, req.ContentLength, cHeaders, cResource)
}

func (s *ABS) initAuth(ctx context.Context, cfg *configpb.ABS) error {
	if cfg.GetAccountKey() != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.GetAccountKey())
		if err != nil {
			return fmt.Errorf("failed to decode account key: %v", err)
		}
		s.key = key
		return nil
	}

	if cfg.GetOauthConfig() != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(cfg.GetOauthConfig(), s.l)
		if err != nil {
			return err
		}
		s.client = oauth2.NewClient(ctx, oauthTS)
		return nil
	}

	// If account key and auth config are not provided, we will use the metadata
	// service to get the bearer token.
	oauthTS, err := oauth.TokenSourceFromConfig(&oauthconfigpb.Config{
		Source: &oauthconfigpb.Config_HttpRequest{
			HttpRequest: &oauthconfigpb.HTTPRequest{
				TokenUrl: proto.String(identityEndpoint),
				Header: map[string]string{
					"Metadata": "true",
				},
				Data: []string{
					"resource=" + storageURL,
					"api-version=" + tokenAPIVersion,
				},
			},
		},
	}, s.l)
	if err != nil {
		return err
	}
	s.client = oauth2.NewClient(ctx, oauthTS)
	return nil
}

func InitABS(ctx context.Context, cfg *configpb.ABS, storagePath string, l *logger.Logger) (*ABS, error) {
	if cfg.GetContainer() == "" {
		return nil, fmt.Errorf("ABS container name is required")
	}

	abs := &ABS{
		container:   cfg.GetContainer(),
		accountName: cfg.GetAccountName(),
		path:        storagePath,
		endpoint:    cfg.GetEndpoint(),
		client:      &http.Client{},
		l:           l,
	}
	if abs.endpoint == "" {
		abs.endpoint = "https://" + abs.accountName + ".blob.core.windows.net"
	}

	return abs, abs.initAuth(ctx, cfg)
}

// signature creates signature for the given string using the account key.
func (s *ABS) signature(stringToSign string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *ABS) uploadRequest(ctx context.Context, content []byte, relPath string) (*http.Request, error) {
	blobPath := path.Join(s.container, s.path, relPath)

	req, err := http.NewRequestWithContext(ctx, "PUT", s.endpoint+"/"+blobPath, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	if req.ContentLength == 0 {
		req.ContentLength = int64(len(content))
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", req.ContentLength))
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", version)
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-date", date)
	return req, nil
}

func (s *ABS) upload(ctx context.Context, r io.Reader, relPath string) error {
	fileContent, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file content: %v", err)
	}

	req, err := s.uploadRequest(ctx, fileContent, relPath)
	if err != nil {
		return err
	}

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
func (s *ABS) Store(ctx context.Context, localPath string, destPathFn func(string) string) error {
	s.l.Infof("Uploading artifacts from %s to: %s", localPath, s.endpoint)

	return walkAndSave(ctx, localPath, destPathFn, func(ctx context.Context, r io.Reader, relPath string) error {
		return s.upload(ctx, r, relPath)
	})
}
