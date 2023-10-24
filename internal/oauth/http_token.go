// Copyright 2023 The Cloudprober Authors.
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
	"io"
	"net/http"
	"time"

	"github.com/cloudprober/cloudprober/internal/httpreq"
	configpb "github.com/cloudprober/cloudprober/internal/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
)

type httpTokenSource struct {
	cache      *tokenCache
	l          *logger.Logger
	httpClient *http.Client
}

func redact(s string) string {
	if len(s) < 50 {
		return s
	}
	return s[0:20] + " ........ " + s[len(s)-20:]
}

func (ts *httpTokenSource) tokenFromHTTP(req *http.Request) (*oauth2.Token, error) {
	if ts.httpClient == nil {
		ts.httpClient = http.DefaultClient
	}

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token URL err: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		s, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token URL response: %v, msg: %s", resp.StatusCode, s)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token URL response: %v", err)
	}
	ts.l.Infof("oauth2: response from token URL: %s", redact(string(respBody)))

	// Parse and verify token
	tok := &jsonToken{}
	if err := json.Unmarshal([]byte(respBody), tok); err != nil {
		return nil, fmt.Errorf("error parsing token URL response (%s) as JSON: %v", redact(string(respBody)), err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("access_token not found in token URL response: %v", tok)
	}
	if tok.ExpiresIn == 0 {
		ts.l.Warningf("oauth2: token's expiration time is not set, we'll renew everytime")
	}

	ts.l.Infof("oauth2: token expires in: %d sec", tok.ExpiresIn)

	return &oauth2.Token{
		AccessToken: tok.AccessToken,
		Expiry:      time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}, nil
}

func newRequest(method, url string, headers map[string]string, data []string) (*http.Request, error) {
	req, err := httpreq.NewRequest(method, url, httpreq.NewRequestBody(data...))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func newHTTPTokenSource(c *configpb.HTTPRequest, refreshExpiryBuffer time.Duration, l *logger.Logger) (oauth2.TokenSource, error) {
	// Verify request parameters are correct.
	_, err := newRequest(c.GetMethod(), c.GetTokenUrl(), c.GetHeader(), c.GetData())
	if err != nil {
		return nil, err
	}

	ts := &httpTokenSource{l: l}

	ts.cache = &tokenCache{
		getToken: func() (*oauth2.Token, error) {
			req, err := newRequest(c.GetMethod(), c.GetTokenUrl(), c.GetHeader(), c.GetData())
			if err != nil {
				return nil, err
			}
			return ts.tokenFromHTTP(req)
		},
		refreshExpiryBuffer: refreshExpiryBuffer,
		l:                   l,
	}
	return ts, nil
}

func (ts *httpTokenSource) Token() (*oauth2.Token, error) {
	return ts.cache.Token()
}
