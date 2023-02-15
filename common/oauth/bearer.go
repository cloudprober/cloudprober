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

	if ts.c.RefreshExpiryBufferSec == nil {
		ts.c.RefreshExpiryBufferSec = proto.Int32(60)
	}

	tok, err := ts.getTokenFromBackend(c)
	if err != nil {
		return nil, err
	}
	ts.cache = &tokenCache{
		tok:                 tok,
		returnCacheOnFail:   true,
		refreshExpiryBuffer: time.Duration(ts.c.GetRefreshExpiryBufferSec()) * time.Second,
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
