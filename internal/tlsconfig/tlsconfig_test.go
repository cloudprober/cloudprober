// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func parseAndVerifyCert(t *testing.T, cert tls.Certificate, wantCommonName string) {
	t.Helper()

	certParsed, err := x509.ParseCertificate(cert.Certificate[0])
	assert.NoError(t, err, "Error parsing TLS certificate")
	assert.Equal(t, wantCommonName, certParsed.Subject.CommonName)
}

func TestUpdateTLSConfig(t *testing.T) {
	cert1 := [2]string{cert1PEM, cert1Key}
	cert2 := [2]string{cert2PEM, cert2Key}

	tempCertF, err := os.CreateTemp("", "cert")
	assert.NoError(t, err, "failed to create temp cert file")
	tempKeyF, err := os.CreateTemp("", "key")
	assert.NoError(t, err, "failed to create temp key file")
	testCert, testKey := tempCertF.Name(), tempKeyF.Name()
	defer os.Remove(testCert)
	defer os.Remove(testKey)

	writeTestCert := func(cert [2]string) {
		assert.NoError(t, os.WriteFile(testCert, []byte(cert[0]), 0644), "failed to write file %s", testCert)
		assert.NoError(t, os.WriteFile(testKey, []byte(cert[1]), 0644), "failed to write file %s", testKey)
	}

	tests := []struct {
		name       string
		serverName string
		baseCert   [2]string
		nextCert   [2]string
		nextKey    []byte
		dynamic    bool
		wantCN     string
		wantNextCN string
		minVersion configpb.TLSVersion
		maxVersion configpb.TLSVersion
		wantMinVer uint16
		wantMaxVer uint16
		wantErr    bool
	}{
		{
			name:     "base-cert1-no-servername",
			baseCert: cert1,
			wantCN:   "cert1.cloudprober.org",
		},
		{
			name:       "base-cert1",
			baseCert:   cert1,
			serverName: "cloudprober.org",
			wantCN:     "cert1.cloudprober.org",
		},
		{
			name:       "base-cert2",
			baseCert:   cert2,
			serverName: "cloudprober.org",
			wantCN:     "cert2.cloudprober.org",
		},
		{
			name:       "base-cert1",
			baseCert:   cert1,
			nextCert:   cert2,
			dynamic:    true,
			serverName: "cloudprober.org",
			wantCN:     "cert1.cloudprober.org",
			wantNextCN: "cert2.cloudprober.org",
		},
		{
			name:       "min-max-version",
			baseCert:   cert1,
			wantCN:     "cert1.cloudprober.org",
			minVersion: configpb.TLSVersion_TLS_1_1,
			maxVersion: configpb.TLSVersion_TLS_1_3,
			wantMinVer: tls.VersionTLS11,
			wantMaxVer: tls.VersionTLS13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := &tls.Config{}
			testConf := &configpb.TLSConfig{
				TlsCertFile:   &testCert,
				TlsKeyFile:    &testKey,
				MinTlsVersion: tt.minVersion.Enum(),
				MaxTlsVersion: tt.maxVersion.Enum(),
			}
			if tt.dynamic {
				testConf.ReloadIntervalSec = proto.Int32(1)
			}
			if tt.serverName != "" {
				testConf.ServerName = &tt.serverName
			}

			writeTestCert(tt.baseCert)

			if err := UpdateTLSConfig(tlsConfig, testConf); (err != nil) != tt.wantErr {
				t.Errorf("UpdateTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.serverName, tlsConfig.ServerName, "ServerName mismatch")
			assert.Equal(t, tt.wantMinVer, tlsConfig.MinVersion, "MinVersion mismatch")
			assert.Equal(t, tt.wantMaxVer, tlsConfig.MaxVersion, "MaxVersion mismatch")

			if !tt.dynamic {
				assert.Nil(t, tlsConfig.GetClientCertificate, "GetClientCertificate should be nil")
				assert.Nil(t, tlsConfig.GetCertificate, "GetCertificate should be nil")
				assert.Len(t, tlsConfig.Certificates, 1, "Certificates should have one entry")
				parseAndVerifyCert(t, tlsConfig.Certificates[0], tt.wantCN)
			}

			if tt.dynamic {
				assert.NotNil(t, tlsConfig.GetClientCertificate, "GetClientCertificate should not be nil")
				assert.NotNil(t, tlsConfig.GetCertificate, "GetCertificate should not be nil")
				assert.Equal(t, 0, len(tlsConfig.Certificates), "Certificates should be empty")

				cert, err := tlsConfig.GetClientCertificate(nil)
				assert.NoError(t, err, "Error getting client TLS certificate")
				parseAndVerifyCert(t, *cert, tt.wantCN)

				cert, err = tlsConfig.GetCertificate(nil)
				assert.NoError(t, err, "Error getting server TLS certificate")
				parseAndVerifyCert(t, *cert, tt.wantCN)

				if tt.nextCert[0] != "" {
					writeTestCert(tt.nextCert)
					time.Sleep(1 * time.Second)

					cert, err := tlsConfig.GetClientCertificate(nil)
					assert.NoError(t, err, "Error getting client TLS certificate")
					parseAndVerifyCert(t, *cert, tt.wantNextCN)

					cert, err = tlsConfig.GetCertificate(nil)
					assert.NoError(t, err, "Error getting server TLS certificate")
					parseAndVerifyCert(t, *cert, tt.wantNextCN)
				}
			}

			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			ts.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
			ts.StartTLS()
			defer ts.Close()

			tlsConfig = tlsConfig.Clone()
			tlsConfig.InsecureSkipVerify = true
			client := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
			res, err := client.Get(ts.URL)
			assert.NoError(t, err)
			res.Body.Close()
			assert.Equal(t, http.StatusOK, res.StatusCode)
		})
	}
}
