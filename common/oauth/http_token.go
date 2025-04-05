// Copyright 2023-2025 The Cloudprober Authors.
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

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/internal/httpreq"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
)

func redact(s string) string {
	if len(s) < 50 {
		return s
	}
	return s[0:20] + " ........ " + s[len(s)-20:]
}

func tokenFromHTTP(client *http.Client, req *http.Request, l *logger.Logger) (*oauth2.Token, error) {
	resp, err := client.Do(req)
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
	l.Infof("oauth2: response from token URL: %s", redact(string(respBody)))

	// Parse and verify token
	tok := &jsonToken{}
	if err := json.Unmarshal([]byte(respBody), tok); err != nil {
		return nil, fmt.Errorf("error parsing token URL response (%s) as JSON: %v", redact(string(respBody)), err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("access_token not found in token URL response: %v", tok)
	}
	if tok.ExpiresIn == 0 {
		l.Warningf("oauth2: token's expiration time is not set, we'll renew everytime")
	}

	l.Infof("oauth2: token expires in: %d sec", tok.ExpiresIn)

	return &oauth2.Token{
		AccessToken: tok.AccessToken,
		Expiry:      time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}, nil
}

func newRequest(c *configpb.HTTPRequest) (*http.Request, error) {
	method, url, headers, data := c.GetMethod(), c.GetTokenUrl(), c.GetHeader(), c.GetData()

	req, err := httpreq.NewRequest(method, url, httpreq.NewRequestBody(data...))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}
