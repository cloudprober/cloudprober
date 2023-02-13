// Copyright 2019 The Cloudprober Authors.
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
	"fmt"
	"sync"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
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

func testTokenFromFile(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetFile() + "_file_token", nil
}

func testTokenFromCmd(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetCmd() + "_cmd_token", nil
}

func testTokenFromGCEMetadata(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetGceServiceAccount() + "_gce_token", nil
}

func TestNewBearerToken(t *testing.T) {
	getTokenFromFile = testTokenFromFile
	getTokenFromCmd = testTokenFromCmd
	getTokenFromGCEMetadata = testTokenFromGCEMetadata

	var tests = []struct {
		config    string
		wantToken string
		noCache   bool
	}{
		{
			config:    "file: \"f\"",
			wantToken: "f_file_token",
		},
		{
			config:    "file: \"f\"\nrefresh_interval_sec: 0",
			wantToken: "f_file_token",
			noCache:   true,
		},
		{
			config:    "cmd: \"c\"",
			wantToken: "c_cmd_token",
		},
		{
			config:    "gce_service_account: \"default\"",
			wantToken: "default_gce_token",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:cache:%v", test.config, !test.noCache), func(t *testing.T) {
			resetCallCounter()
			testC := &configpb.BearerToken{}
			assert.NoError(t, proto.UnmarshalText(test.config, testC), "error parsing test config")

			// Call counter should always increase during token source creation.
			expectedC := callCounter() + 1
			cts, err := newBearerTokenSource(testC, nil)
			assert.NoError(t, err, "error while creating new token source")
			assert.Equal(t, expectedC, callCounter(), "unexpected call counter (1st call)")

			// Get token again
			if test.noCache {
				expectedC++
			}
			tok, err := cts.Token()
			assert.NoError(t, err, "error getting token")
			assert.Equal(t, test.wantToken, tok.AccessToken, "Token mismatch")
			assert.Equal(t, expectedC, callCounter(), "unexpected call counter (2nd call)")
		})
	}
}

var (
	calledTestTokenOnce   bool
	calledTestTokenOnceMu sync.Mutex
)

func testTokenRefresh(c *configpb.BearerToken) (string, error) {
	calledTestTokenOnceMu.Lock()
	defer calledTestTokenOnceMu.Unlock()
	if calledTestTokenOnce {
		return "new-token", nil
	}
	calledTestTokenOnce = true
	return "old-token", nil
}

// TestRefreshCycle verifies that token gets refreshed after the refresh
// cycle.
func TestRefreshCycle(t *testing.T) {
	getTokenFromCmd = testTokenRefresh
	// Disable caching by setting refresh_interval_sec to 0.
	testConfig := "cmd: \"c\"\nrefresh_interval_sec: 1"

	testC := &configpb.BearerToken{}
	err := proto.UnmarshalText(testConfig, testC)
	if err != nil {
		t.Fatalf("error parsing test config (%s): %v", testConfig, err)
	}

	ts, err := newBearerTokenSource(testC, nil)
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		t.Errorf("unexpected error while retrieving token from config (%s): %v", testConfig, err)
	}

	oldToken := "old-token"
	newToken := "new-token"

	if tok.AccessToken != oldToken {
		t.Errorf("ts.Token(): got=%s, expected=%s", tok, oldToken)
	}

	time.Sleep(5 * time.Second)

	tok, err = ts.Token()
	if err != nil {
		t.Errorf("unexpected error while retrieving token from config (%s): %v", testConfig, err)
	}

	if tok.AccessToken != newToken {
		t.Errorf("ts.Token(): got=%s, expected=%s", tok, newToken)
	}
}
