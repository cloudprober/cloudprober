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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/internal/file"
	configpb "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
)

type cacheEntry struct {
	cert       *tls.Certificate
	lastReload time.Time
}

var global = struct {
	cache map[[2]string]cacheEntry
	mu    sync.RWMutex
}{
	cache: make(map[[2]string]cacheEntry),
}

var tlsVersionMap = map[configpb.TLSVersion]uint16{
	configpb.TLSVersion_TLS_1_0: tls.VersionTLS10,
	configpb.TLSVersion_TLS_1_1: tls.VersionTLS11,
	configpb.TLSVersion_TLS_1_2: tls.VersionTLS12,
	configpb.TLSVersion_TLS_1_3: tls.VersionTLS13,
}

func loadCert(certFile, keyFile string) (*tls.Certificate, error) {
	certPEMBlock, err := file.ReadFile(context.Background(), certFile)
	if err != nil {
		return nil, fmt.Errorf("common/tlsconfig: error reading TLS cert file (%s): %v", certFile, err)
	}

	keyPEMBlock, err := file.ReadFile(context.Background(), keyFile)
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
		caCert, err := file.ReadFile(context.Background(), c.GetCaCertFile())
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
		certF, keyF := c.GetTlsCertFile(), c.GetTlsKeyFile()

		// Even if we are live-reloading the cert, verify early that we can
		// load the certificate.
		cert, err := loadCert(certF, keyF)
		if err != nil {
			return err
		}

		if c.GetReloadIntervalSec() > 0 {
			key := [2]string{certF, keyF}

			reloadCertIfNeeded := func() (*tls.Certificate, error) {
				global.mu.RLock()
				entry, ok := global.cache[key]
				global.mu.RUnlock()

				if ok && (time.Since(entry.lastReload) < time.Duration(c.GetReloadIntervalSec())*time.Second) {
					return entry.cert, nil
				}

				cert, err := loadCert(certF, keyF)
				if err != nil {
					return nil, err
				}

				global.mu.Lock()
				global.cache[key] = cacheEntry{
					cert:       cert,
					lastReload: time.Now(),
				}
				global.mu.Unlock()

				return cert, nil
			}

			tlsConfig.GetClientCertificate = func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return reloadCertIfNeeded()
			}
			tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return reloadCertIfNeeded()
			}
		} else {
			tlsConfig.Certificates = append(tlsConfig.Certificates, *cert)
		}
	}

	if c.GetServerName() != "" {
		tlsConfig.ServerName = c.GetServerName()
	}

	if c.GetMinTlsVersion() != configpb.TLSVersion_TLS_AUTO {
		tlsConfig.MinVersion = tlsVersionMap[c.GetMinTlsVersion()]
	}
	if c.GetMaxTlsVersion() != configpb.TLSVersion_TLS_AUTO {
		tlsConfig.MaxVersion = tlsVersionMap[c.GetMaxTlsVersion()]
	}

	return nil
}
