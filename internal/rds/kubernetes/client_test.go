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

package kubernetes

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	tlsconfigpb "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
	cpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var testCACert = `
-----BEGIN CERTIFICATE-----
MIICVjCCAb+gAwIBAgIUY0TRq/rPKnOpZ+Bbv9hBMlKgiP0wDQYJKoZIhvcNAQEL
BQAwPTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMSEwHwYDVQQKDBhJbnRlcm5l
dCBXaWRnaXRzIFB0eSBMdGQwHhcNMTkxMjE4MDA1MDMxWhcNMjAwMTE3MDA1MDMx
WjA9MQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1z5z
InMY7e5AOgYnq9ACqwtIYRZZwEbBSy57Pe9kftEpUy5wfdN5YtBEiW+juy6CZLns
WKQ3sZ/gnbYvdRfkHnbTU6DZdt741H/YXMnbkT28In1NJYvz/FiJWiphUxbXNEmi
laHAbX+zkOnL81LX7NAArKlk0biK8iglW80oeRMCAwEAAaNTMFEwHQYDVR0OBBYE
FDHgRfu3kXFxPj0f4XYsxxukRYeKMB8GA1UdIwQYMBaAFDHgRfu3kXFxPj0f4XYs
xxukRYeKMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADgYEAI4xvCBkE
NbrbrZ4rPllcmYxgBrTvOUbaG8Lmhx1/qoOyx2LCca0pMQGB8cyMtT5/D7IZUlHk
5IG8ts/LpaDbRhTP4MQUCoN/FgAJ4e5Y33VWJocucgEFyv4aKl0Xgg+hO4ejpaxr
JeDPlMt+8OTQNVC63+SPzOvlUsOLFX74WMo=
-----END CERTIFICATE-----
`

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

func TestNewClientWithNoOauth(t *testing.T) {
	cacrtF := testFileWithContent(t, testCACert)
	defer os.Remove(cacrtF) // clean up

	oldLocalCACert := LocalCACert
	LocalCACert = cacrtF
	defer func() { LocalCACert = oldLocalCACert }()

	testToken := "test-token"
	tokenF := testFileWithContent(t, testToken)
	defer os.Remove(tokenF) // clean up

	os.Setenv("KUBERNETES_SERVICE_HOST", "test-api-host")
	os.Setenv("KUBERNETES_SERVICE_PORT", "4123")

	tc := &client{
		cfg: &cpb.ProviderConfig{},
	}
	transport, err := tc.httpTransportWithTLS()
	assert.NoError(t, err, "error creating client")
	assert.NotNil(t, transport.TLSClientConfig, "TLS config")

	err = tc.initAPIHost()
	assert.NoError(t, err, "error initialize API host")
	assert.Equal(t, "test-api-host:4123", tc.apiHost, "k8s API host")
}

func TestNewClientWithTLS(t *testing.T) {
	cacrtF := testFileWithContent(t, testCACert)
	defer os.Remove(cacrtF) // clean up

	oldLocalCACert := LocalCACert
	LocalCACert = cacrtF
	defer func() { LocalCACert = oldLocalCACert }()

	testAPIServerAddr := "test-api-server-addr"
	tc, err := newClientWithoutToken(&cpb.ProviderConfig{
		ApiServerAddress: &testAPIServerAddr,
		TlsConfig: &tlsconfigpb.TLSConfig{
			CaCertFile: proto.String(cacrtF),
		},
	}, nil)
	if err != nil {
		t.Fatalf("error while creating new client: %v", err)
	}

	if tc.apiHost != testAPIServerAddr {
		t.Errorf("client.apiHost: got=%s, exepcted=%s", tc.apiHost, testAPIServerAddr)
	}

	if tc.bearer != "" {
		t.Errorf("client.bearer: got=%s, exepcted=''", tc.bearer)
	}

	if tc.httpC == nil || tc.httpC.Transport.(*http.Transport).TLSClientConfig == nil {
		t.Errorf("Client's HTTP client or TLS config are unexpectedly nil.")
	}
}

func TestClientHTTPRequest(t *testing.T) {
	tests := []struct {
		name          string
		labelselector []string
		wantURL       string
	}{
		{
			name:    "no-label",
			wantURL: "https://test-api-host/api/v1/pods",
		},
		{
			name:          "with-label",
			labelselector: []string{"app=cloudprober"},
			wantURL:       "https://test-api-host/api/v1/pods?labelSelector=app%3Dcloudprober",
		},
		{
			name:          "with-two-labels",
			labelselector: []string{"app=cloudprober", "env!=dev"},
			wantURL:       "https://test-api-host/api/v1/pods?labelSelector=app%3Dcloudprober%2Cenv%21%3Ddev",
		},
	}

	testAPIHost := "test-api-host"
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &client{
				apiHost: testAPIHost,
				cfg: &cpb.ProviderConfig{
					LabelSelector: test.labelselector,
				},
			}
			req, err := tc.httpRequest("api/v1/pods")
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}

			assert.Equal(t, test.wantURL, req.URL.String())
		})
	}

}
