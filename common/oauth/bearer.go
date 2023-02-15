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
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/file"
	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/protobuf/proto"
)

type bearerTokenSource struct {
	c                   *configpb.BearerToken
	getTokenFromBackend func(*configpb.BearerToken) (*oauth2.Token, error)
	cache               *oauth2.Token
	mu                  sync.RWMutex
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

var getTokenFromFile = func(c *configpb.BearerToken) (*oauth2.Token, error) {
	b, err := file.ReadFile(c.GetFile())
	if err != nil {
		return nil, err
	}
	return bytesToToken(b), nil
}

var getTokenFromCmd = func(c *configpb.BearerToken) (*oauth2.Token, error) {
	var cmd *exec.Cmd

	cmdParts := strings.Split(c.GetCmd(), " ")
	cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", cmd)
	}

	return bytesToToken(out), nil
}

var getTokenFromGCEMetadata = func(c *configpb.BearerToken) (*oauth2.Token, error) {
	ts := google.ComputeTokenSource(c.GetGceServiceAccount())
	tok, err := ts.Token()
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func newBearerTokenSource(c *configpb.BearerToken, l *logger.Logger) (oauth2.TokenSource, error) {
	ts := &bearerTokenSource{
		c: c,
		l: l,
	}

	switch ts.c.Source.(type) {
	case *configpb.BearerToken_File:
		ts.getTokenFromBackend = getTokenFromFile

	case *configpb.BearerToken_Cmd:
		ts.getTokenFromBackend = getTokenFromCmd

	case *configpb.BearerToken_GceServiceAccount:
		ts.getTokenFromBackend = getTokenFromGCEMetadata

	default:
		ts.getTokenFromBackend = getTokenFromGCEMetadata
	}

	tok, err := ts.getTokenFromBackend(c)
	if err != nil {
		return nil, err
	}
	ts.cache = tok

	// With the move to proto3, set default value explicitly.
	if ts.c.RefreshIntervalSec == nil {
		ts.c.RefreshIntervalSec = proto.Float32(30)
	}

	if ts.c.GetRefreshIntervalSec() == 0 {
		return ts, nil
	}

	go func() {
		interval := time.Duration(ts.c.GetRefreshIntervalSec()) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			// If we've been getting JSON token with non-zero expiry, skip
			// refreshing periodically.
			if ts.cache != nil && !ts.cache.Expiry.IsZero() {
				return
			}
			tok, err := ts.getTokenFromBackend(ts.c)

			if err != nil {
				ts.l.Warningf("oauth.bearerTokenSource: %s", err)
				continue
			}

			ts.mu.Lock()
			ts.cache = tok
			ts.mu.Unlock()
		}
	}()

	return ts, nil
}

func (ts *bearerTokenSource) setCache(tok *oauth2.Token) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.cache = tok
}

func (ts *bearerTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.RLock()
	tok := ts.cache
	ts.mu.RUnlock()

	if tok != nil && time.Until(tok.Expiry) > time.Duration(ts.c.GetRefreshExpiryBufferSec())*time.Second {
		return tok, nil
	}

	tok, err := ts.getTokenFromBackend(ts.c)
	if err != nil {
		if tok != nil {
			ts.l.Errorf("oauth.bearerTokenSource: failed to refresh the token: %v, returning stale token", err)
			return ts.cache, nil
		}
		return nil, err
	}
	ts.setCache(tok)
	return tok, nil
}
