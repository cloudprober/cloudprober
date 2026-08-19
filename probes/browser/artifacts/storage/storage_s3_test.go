// Copyright 2026 The Cloudprober Authors.
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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
	"github.com/stretchr/testify/assert"
)

func TestInitS3WithWebIdentity(t *testing.T) {
	var mu sync.Mutex
	var stsRequest, headBucketRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() failed: %v", err)
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			if got := r.Form.Get("Action"); got != "AssumeRoleWithWebIdentity" {
				t.Errorf("STS action = %q, want AssumeRoleWithWebIdentity", got)
			}
			stsRequest = true
			w.Header().Set("Content-Type", "text/xml")
			_, err := fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleWithWebIdentityResult><Credentials>
    <AccessKeyId>test-access-key</AccessKeyId><SecretAccessKey>test-secret-key</SecretAccessKey>
    <SessionToken>test-session-token</SessionToken><Expiration>2035-01-01T00:00:00Z</Expiration>
  </Credentials></AssumeRoleWithWebIdentityResult>
  <ResponseMetadata><RequestId>test-request</RequestId></ResponseMetadata>
</AssumeRoleWithWebIdentityResponse>`)
			assert.NoError(t, err)
			return
		}

		if r.Method == http.MethodHead && strings.Contains(r.URL.Path, "test-bucket") {
			headBucketRequest = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))
	defer server.Close()

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("fake-web-identity-token"), 0600); err != nil {
		t.Fatalf("os.WriteFile() failed: %v", err)
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/test-role")
	t.Setenv("AWS_ROLE_SESSION_NAME", "cloudprober-test")
	t.Setenv("AWS_ENDPOINT_URL_STS", server.URL)
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	bucket := "test-bucket"
	region := "us-east-1"
	endpoint := server.URL
	_, err := InitS3(context.Background(), &configpb.S3{
		Bucket:   &bucket,
		Region:   &region,
		Endpoint: &endpoint,
	}, "artifacts", nil)
	if err != nil {
		t.Fatalf("InitS3() failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !stsRequest {
		t.Error("InitS3() did not request web identity credentials from STS")
	}
	if !headBucketRequest {
		t.Error("InitS3() did not perform the startup HeadBucket request")
	}
}
