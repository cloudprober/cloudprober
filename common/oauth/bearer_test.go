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
	"errors"
	"io/ioutil"
	"os"
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
		if strings.HasSuffix(c.GetFile(), "fail") {
			return nil, errors.New("failed_reading_token")
		}
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
	if strings.Contains(c.GetCmd(), "_exp_") {
		a := strings.Split(c.GetCmd(), "_")
		d, _ := time.ParseDuration(a[len(a)-1])
		exp = time.Now().Add(d)
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

func testK8SToken(c *configpb.BearerToken) (*oauth2.Token, error) {
	suffix := ""
	if callCounter() > 0 {
		suffix = "_new"
	}
	incCallCounter()
	return &oauth2.Token{AccessToken: "k8s_token" + suffix, Expiry: time.Now().Add(time.Hour)}, nil
}

func TestNewBearerToken(t *testing.T) {
	oldTokenFunctions := tokenFunctions
	tokenFunctions.fromFile = testTokenFromFile
	tokenFunctions.fromCmd = testTokenFromCmd
	tokenFunctions.fromGCEMetadata = testTokenFromGCEMetadata
	tokenFunctions.fromK8sTokenFile = testK8SToken

	defer func() {
		tokenFunctions = oldTokenFunctions
	}()

	var tests = []struct {
		name         string
		config       string
		wantToken    string
		wantErr      bool
		wait         time.Duration
		wantNewToken bool
	}{
		{
			config:    "file: \"f_json\"",
			wantToken: "f_json_file_token",
		},
		{
			config:    "cmd: \"c\"",
			wantToken: "c_cmd_token",
		},
		{
			config:    "gce_service_account: \"default\"",
			wantToken: "default_gce_token",
		},
		{
			config:    "k8s_local_token: true",
			wantToken: "k8s_token",
		},
		{
			config:  "k8s_local_token: false",
			wantErr: true,
		},
		{
			name:         "Refresh in 1s",
			config:       "file: \"f\"\nrefresh_interval_sec: 1",
			wantToken:    "f_file_token",
			wait:         3 * time.Second,
			wantNewToken: true,
		},
		{
			name:         "Refresh fails, go with the cache",
			config:       "file: \"f_fail\"\nrefresh_interval_sec: 1",
			wantToken:    "f_fail_file_token",
			wait:         3 * time.Second,
			wantNewToken: false,
		},
		{
			name:         "JSON from file, ignore refresh interval",
			config:       "file: \"f_json\"\nrefresh_interval_sec: 1",
			wantToken:    "f_json_file_token",
			wait:         3 * time.Second,
			wantNewToken: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name+":"+test.config, func(t *testing.T) {
			resetCallCounter()
			testC := &configpb.BearerToken{}
			assert.NoError(t, prototext.Unmarshal([]byte(test.config), testC), "error parsing test config")

			testRefreshExpiryBuffer := 10 * time.Second

			// Call counter should always increase during token source creation.
			expectedC := callCounter() + 1
			cts, err := newBearerTokenSource(testC, testRefreshExpiryBuffer, nil)
			if (err != nil) != test.wantErr {
				t.Errorf("newBearerTokenSource() error = %v, wantErr %v", err, test.wantErr)
			}
			if err != nil {
				return
			}

			// verify token cache
			tc := cts.(*bearerTokenSource).cache
			assert.Equal(t, testRefreshExpiryBuffer, tc.refreshExpiryBuffer, "token cache refresh expiry buffer")
			assert.Equal(t, tc.ignoreExpiryIfZero, true)

			assert.Equal(t, expectedC, callCounter(), "unexpected call counter (1st call)")

			// Get token again
			if test.wantNewToken {
				time.Sleep(test.wait) // Wait for refresh
				test.wantToken += "_new"
			}
			tok, err := cts.Token()
			assert.NoError(t, err, "error getting token")
			assert.Equal(t, test.wantToken, tok.AccessToken, "Token mismatch")
		})
	}
}

func testFileWithContent(t *testing.T, content string) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file for testing: %v", err)
		return ""
	}

	if _, err := f.Write([]byte(content)); err != nil {
		os.Remove(f.Name()) // clean up
		t.Fatalf("Error writing %s to temporary file: %s", content, f.Name())
		return ""
	}

	return f.Name()
}

func TestK8STokenSource(t *testing.T) {
	tests := []struct {
		testToken string
		want      string
		badFile   bool
		wantErr   bool
	}{
		{
			testToken: "test-token",
			want:      "test-token",
		},
		{
			testToken: "error",
			badFile:   true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testToken, func(t *testing.T) {
			tokenF := testFileWithContent(t, tt.testToken)
			defer os.Remove(tokenF) // clean up

			if tt.badFile {
				tokenF = tokenF + "__random__bad_path__"
			}

			oldK8STokenFile := k8sTokenFile
			k8sTokenFile = tokenF
			defer func() { k8sTokenFile = oldK8STokenFile }()

			ts, err := K8STokenSource(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("K8STokenSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotToken, _ := ts.Token()
			assert.Equal(t, tt.testToken, gotToken.AccessToken, "access token")
		})
	}
}
