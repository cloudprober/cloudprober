// Copyright 2019-2023 The Cloudprober Authors.
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

type bearerTokenSource struct {
	c                   *configpb.BearerToken
	getTokenFromBackend func(*configpb.BearerToken) (*oauth2.Token, error)
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

var tokenFunctions = struct {
	fromFile, fromCmd, fromGCEMetadata, fromK8sTokenFile func(c *configpb.BearerToken) (*oauth2.Token, error)
}{
	fromFile: func(c *configpb.BearerToken) (*oauth2.Token, error) {
		b, err := file.ReadFile(context.Background(), c.GetFile())
		if err != nil {
			return nil, err
		}
		return bytesToToken(b), nil
	},
	fromCmd: func(c *configpb.BearerToken) (*oauth2.Token, error) {
		var cmd *exec.Cmd

		cmdParts := strings.Split(c.GetCmd(), " ")
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to execute command: %v", cmd)
		}

		return bytesToToken(out), nil
	},
	fromGCEMetadata: func(c *configpb.BearerToken) (*oauth2.Token, error) {
		ts := google.ComputeTokenSource(c.GetGceServiceAccount())
		tok, err := ts.Token()
		if err != nil {
			return nil, err
		}
		return tok, nil
	},
	fromK8sTokenFile: func(c *configpb.BearerToken) (*oauth2.Token, error) {
		b, err := os.ReadFile(k8sTokenFile)
		if err != nil {
			return nil, err
		}
		return &oauth2.Token{AccessToken: string(b)}, nil
	},
}

func newBearerTokenSource(c *configpb.BearerToken, refreshExpiryBuffer time.Duration, l *logger.Logger) (oauth2.TokenSource, error) {
	ts := &bearerTokenSource{
		c: c,
		l: l,
	}

	var tokenBackendFunc func(*configpb.BearerToken) (*oauth2.Token, error)

	switch ts.c.Source.(type) {
	case *configpb.BearerToken_File:
		tokenBackendFunc = tokenFunctions.fromFile

	case *configpb.BearerToken_Cmd:
		tokenBackendFunc = tokenFunctions.fromCmd

	case *configpb.BearerToken_GceServiceAccount:
		tokenBackendFunc = tokenFunctions.fromGCEMetadata

	case *configpb.BearerToken_K8SLocalToken:
		if !c.GetK8SLocalToken() {
			return nil, fmt.Errorf("k8s_local_token cannot be false, config: <%v>", c.String())
		}
		tokenBackendFunc = tokenFunctions.fromK8sTokenFile

	default:
		return nil, fmt.Errorf("unknown source: %v", ts.c.GetSource())
	}

	ts.getTokenFromBackend = func(c *configpb.BearerToken) (*oauth2.Token, error) {
		l.Debugf("oauth.bearerTokenSource: Getting a new token using config: %s", c.String())
		return tokenBackendFunc(c)
	}

	tok, err := ts.getTokenFromBackend(c)
	if err != nil {
		return nil, err
	}
	ts.cache = &tokenCache{
		tok:                 tok,
		refreshExpiryBuffer: refreshExpiryBuffer,
		ignoreExpiryIfZero:  true,
		getToken:            func() (*oauth2.Token, error) { return ts.getTokenFromBackend(c) },
		l:                   l,
	}

	// For JSON tokens return now.
	if tok != nil && !tok.Expiry.IsZero() {
		return ts, nil
	}

	// Default refresh interval
	if ts.c.RefreshIntervalSec == nil {
		ts.c.RefreshIntervalSec = proto.Float32(30)
	}
	// Explicitly set to zero, so no refreshing.
	if ts.c.GetRefreshIntervalSec() == 0 {
		return ts, nil
	}

	go func() {
		interval := time.Duration(ts.c.GetRefreshIntervalSec()) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			tok, err := ts.getTokenFromBackend(ts.c)

			if err != nil {
				ts.l.Warningf("oauth.bearerTokenSource: %s", err)
				continue
			}

			ts.cache.setToken(tok)
		}
	}()

	return ts, nil
}

func (ts *bearerTokenSource) Token() (*oauth2.Token, error) {
	return ts.cache.Token()
}

func K8STokenSource(l *logger.Logger) (oauth2.TokenSource, error) {
	return newBearerTokenSource(&configpb.BearerToken{
		Source: &configpb.BearerToken_K8SLocalToken{
			K8SLocalToken: true,
		},
		RefreshIntervalSec: proto.Float32(60),
	}, time.Minute, l)
}
