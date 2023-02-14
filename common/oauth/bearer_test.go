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
	"strings"
	"sync"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/prototext"
)

var global struct {
	callCounter int
	mu          sync.RWMutex
}

func resetCallCounter() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.callCounter = 0
}

func incCallCounter() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.callCounter++
}

func callCounter() int {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.callCounter
}

func testTokenFromFile(c *configpb.BearerToken) (*oauth2.Token, error) {
	suffix := ""
	if callCounter() > 0 {
		suffix = "_new"
	}
	incCallCounter()
	exp := time.Time{}
	if strings.HasSuffix(c.GetFile(), "json") {
		exp = time.Now().Add(time.Hour)
	}

	return &oauth2.Token{AccessToken: c.GetFile() + "_file_token" + suffix, Expiry: exp}, nil
}

func testTokenFromCmd(c *configpb.BearerToken) (*oauth2.Token, error) {
	suffix := ""
	if callCounter() > 0 {
		suffix = "_new"
	}
	incCallCounter()
	exp := time.Now().Add(time.Hour)
	if strings.HasSuffix(c.GetCmd(), "lowexp") {
		exp = time.Now().Add(time.Millisecond)
	}
	return &oauth2.Token{AccessToken: c.GetCmd() + "_cmd_token" + suffix, Expiry: exp}, nil
}

func testTokenFromGCEMetadata(c *configpb.BearerToken) (*oauth2.Token, error) {
	suffix := ""
	if callCounter() > 0 {
		suffix = "_new"
	}
	incCallCounter()
	return &oauth2.Token{AccessToken: c.GetGceServiceAccount() + "_gce_token" + suffix, Expiry: time.Now().Add(time.Hour)}, nil
}

func TestNewBearerToken(t *testing.T) {
	getTokenFromFile = testTokenFromFile
	getTokenFromCmd = testTokenFromCmd
	getTokenFromGCEMetadata = testTokenFromGCEMetadata

	var tests = []struct {
		config       string
		wantToken    string
		wantNewToken bool
	}{
		{
			config:    "file: \"f_json\"",
			wantToken: "f_json_file_token",
		},
		{
			config:       "file: \"f\"\nrefresh_interval_sec: 1",
			wantToken:    "f_file_token",
			wantNewToken: true, // refresh in 1s.
		},
		{
			config:       "file: \"f_json\"\nrefresh_interval_sec: 1",
			wantToken:    "f_json_file_token",
			wantNewToken: false, // refresh interval is ignored for json
		},
		{
			config:    "cmd: \"c\"",
			wantToken: "c_cmd_token",
		},
		{
			config:       "cmd: \"c_lowexp\"",
			wantToken:    "c_lowexp_cmd_token",
			wantNewToken: true,
		},
		{
			config:    "gce_service_account: \"default\"",
			wantToken: "default_gce_token",
		},
	}

	for _, test := range tests {
		t.Run(test.config, func(t *testing.T) {
			resetCallCounter()
			testC := &configpb.BearerToken{}
			assert.NoError(t, prototext.Unmarshal([]byte(test.config), testC), "error parsing test config")

			// Call counter should always increase during token source creation.
			expectedC := callCounter() + 1
			cts, err := newBearerTokenSource(testC, nil)
			assert.NoError(t, err, "error while creating new token source")
			assert.Equal(t, expectedC, callCounter(), "unexpected call counter (1st call)")

			// Get token again
			if test.wantNewToken {
				time.Sleep(5 * time.Second) // Wait for refresh
				test.wantToken += "_new"
			}
			tok, err := cts.Token()
			assert.NoError(t, err, "error getting token")
			assert.Equal(t, test.wantToken, tok.AccessToken, "Token mismatch")
		})
	}
}
