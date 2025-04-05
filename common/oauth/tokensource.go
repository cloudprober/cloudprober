// Copyright 2019-2025 The Cloudprober Authors.
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

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/internal/file"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/protobuf/proto"
)

var k8sTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
var httpClient = http.DefaultClient

type genericTokenSource struct {
	c                   *configpb.Config
	getTokenFromBackend func(*configpb.Config) (*oauth2.Token, error)
	cache               *tokenCache
	l                   *logger.Logger
}

func bytesToToken(b []byte) *oauth2.Token {
	tok := &jsonToken{}
	err := json.Unmarshal(b, tok)
	if err != nil {
		return &oauth2.Token{AccessToken: string(b)}
	}
	return &oauth2.Token{
		AccessToken: tok.AccessToken,
		Expiry:      time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}
}

func readFromFile(c *configpb.Config) (*oauth2.Token, error) {
	b, err := file.ReadFile(context.Background(), c.GetFile())
	if err != nil {
		return nil, err
	}
	return bytesToToken(b), nil
}

func readFromCommand(c *configpb.Config) (*oauth2.Token, error) {
	var cmd *exec.Cmd

	cmdParts := strings.Split(c.GetCmd(), " ")
	cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", cmd)
	}

	return bytesToToken(out), nil
}

var tokenFunctions = struct {
	fromFile, fromCmd, fromGCEMetadata, fromK8sTokenFile, fromHTTP func(c *configpb.Config) (*oauth2.Token, error)
}{
	fromFile: readFromFile,
	fromCmd:  readFromCommand,
	fromGCEMetadata: func(c *configpb.Config) (*oauth2.Token, error) {
		ts := google.ComputeTokenSource(c.GetGceServiceAccount())
		tok, err := ts.Token()
		if err != nil {
			return nil, err
		}
		return tok, nil
	},
	fromK8sTokenFile: func(c *configpb.Config) (*oauth2.Token, error) {
		b, err := os.ReadFile(k8sTokenFile)
		if err != nil {
			return nil, err
		}
		return &oauth2.Token{AccessToken: string(b)}, nil
	},
	fromHTTP: func(c *configpb.Config) (*oauth2.Token, error) {
		hc := c.GetHttpRequest()
		req, err := newRequest(hc)
		if err != nil {
			return nil, err
		}
		return tokenFromHTTP(httpClient, req, &logger.Logger{})
	},
}

func (ts *genericTokenSource) verifyTokenSource(c *configpb.Config) (*oauth2.Token, error) {
	// For HTTP request, only verify that request parameters are correct.
	if c.GetHttpRequest() != nil {
		_, err := newRequest(c.GetHttpRequest())
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return ts.getTokenFromBackend(c)
}

func (ts *genericTokenSource) refreshOnInterval(initToken *oauth2.Token) bool {
	if ts.c.GetRefreshIntervalSec() == 0 {
		return false
	}

	// Set explicitly to non-zero value, we want to refresh regardless
	if ts.c.RefreshIntervalSec != nil {
		return true
	}

	// If we got a non-nil token but it doesn't have expiry, we refresh at
	// default interval
	if initToken != nil && initToken.Expiry.IsZero() {
		return true
	}

	// This should handle the http request case, where we don't have initial
	// token, but we don't want to refresh on interval as well, unless
	// explicitly specified (covered in the first case).
	return false
}

func newTokenSource(c *configpb.Config, refreshExpiryBuffer time.Duration, l *logger.Logger) (oauth2.TokenSource, error) {
	ts := &genericTokenSource{
		c: c,
		l: l,
	}

	switch c.Source.(type) {
	case *configpb.Config_File:
		ts.getTokenFromBackend = tokenFunctions.fromFile

	case *configpb.Config_HttpRequest:
		ts.getTokenFromBackend = tokenFunctions.fromHTTP

	case *configpb.Config_Cmd:
		ts.getTokenFromBackend = tokenFunctions.fromCmd

	case *configpb.Config_GceServiceAccount:
		ts.getTokenFromBackend = tokenFunctions.fromGCEMetadata

	case *configpb.Config_K8SLocalToken:
		if !c.GetK8SLocalToken() {
			return nil, fmt.Errorf("k8s_local_token cannot be false, config: <%v>", c.String())
		}
		ts.getTokenFromBackend = tokenFunctions.fromK8sTokenFile

	default:
		return nil, fmt.Errorf("unknown source: %v", c.Source)
	}

	ts.cache = &tokenCache{
		refreshExpiryBuffer: refreshExpiryBuffer,
		getToken:            func() (*oauth2.Token, error) { return ts.getTokenFromBackend(c) },
		l:                   l,
	}

	// Verify and get initial token if available.
	tok, err := ts.verifyTokenSource(c)
	if err != nil {
		return nil, err
	}
	ts.cache.tok = tok

	if !ts.refreshOnInterval(tok) {
		return ts, nil
	}

	// Since we're refreshing at regular interval, disable refreshing in cache.
	ts.cache.refreshingOnInterval = true

	go func() {
		interval := time.Duration(ts.c.GetRefreshIntervalSec()) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			tok, err := ts.getTokenFromBackend(ts.c)

			if err != nil {
				ts.l.Warningf("oauth.genericTokenSource: %s", err)
				continue
			}

			ts.cache.setToken(tok)
		}
	}()

	return ts, nil
}

func (ts *genericTokenSource) Token() (*oauth2.Token, error) {
	return ts.cache.Token()
}

func K8STokenSource(l *logger.Logger) (oauth2.TokenSource, error) {
	return newTokenSource(&configpb.Config{
		Source: &configpb.Config_K8SLocalToken{
			K8SLocalToken: true,
		},
		RefreshIntervalSec: proto.Float32(60),
	}, time.Minute, l)
}
