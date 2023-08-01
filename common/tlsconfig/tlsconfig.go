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

// Package tlsconfig implements utilities to parse TLSConfig.
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/cloudprober/cloudprober/common/file"
	configpb "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
)

func loadCert(certFile, keyFile string, refreshInterval time.Duration) (*tls.Certificate, error) {
	readFile := file.ReadFile
	if refreshInterval > 0 {
		readFile = func(filename string) ([]byte, error) {
			return file.ReadWithCache(filename, refreshInterval)
		}
	}

	certPEMBlock, err := readFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("common/tlsconfig: error reading TLS cert file (%s): %v", certFile, err)
	}

	keyPEMBlock, err := readFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("common/tlsconfig: error reading TLS key file (%s): %v", keyFile, err)
	}

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	return &cert, err
}

// UpdateTLSConfig parses the provided protobuf and updates the tls.Config object.
func UpdateTLSConfig(tlsConfig *tls.Config, c *configpb.TLSConfig) error {
	if c.GetDisableCertValidation() {
		tlsConfig.InsecureSkipVerify = true
	}

	if c.GetCaCertFile() != "" {
		caCert, err := file.ReadFile(c.GetCaCertFile())
		if err != nil {
			return fmt.Errorf("common/tlsconfig: error reading CA cert file (%s): %v", c.GetCaCertFile(), err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("error while adding CA certs from: %s", c.GetCaCertFile())
		}

		tlsConfig.RootCAs = caCertPool
	}

	if c.GetTlsCertFile() != "" {
		// Even if we are live-reloading the cert, verify early that we can
		// load the certificate.
		cert, err := loadCert(c.GetTlsCertFile(), c.GetTlsKeyFile(), 0)
		if err != nil {
			return err
		}

		if c.GetReloadIntervalSec() > 0 {
			tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return loadCert(c.GetTlsCertFile(), c.GetTlsKeyFile(), time.Duration(c.GetReloadIntervalSec())*time.Second)
			}
			return nil
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, *cert)
	}

	if c.GetServerName() != "" {
		tlsConfig.ServerName = c.GetServerName()
	}

	return nil
}
