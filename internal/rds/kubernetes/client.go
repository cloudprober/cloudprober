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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/oauth"
	configpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
)

// Variables defined by Kubernetes spec to find out local CA cert and token.
var (
	LocalCACert    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	LocalTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// client encapsulates an in-cluster kubeapi client.
type client struct {
	cfg     *configpb.ProviderConfig
	httpC   *http.Client
	apiHost string
	bearer  string
	l       *logger.Logger
}

func (c *client) httpRequest(url string) (*http.Request, error) {
	url = fmt.Sprintf("https://%s/%s", c.apiHost, strings.TrimPrefix(url, "/"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.bearer != "" {
		req.Header.Add("Authorization", c.bearer)
	}

	if len(c.cfg.GetLabelSelector()) != 0 {
		values := req.URL.Query()
		values.Add("labelSelector", strings.Join(c.cfg.GetLabelSelector(), ","))
		req.URL.RawQuery = values.Encode()
	}

	return req, nil
}

func (c *client) getURL(url string) ([]byte, error) {
	req, err := c.httpRequest(url)
	if err != nil {
		return nil, err
	}

	c.l.Debugf("kubernetes.client: getting URL: %s", req.URL.String())
	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP response status code: %d, status: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *client) initAPIHost() error {
	c.apiHost = c.cfg.GetApiServerAddress()
	if c.apiHost != "" {
		return nil
	}

	// If API server address is not given, assume in-cluster operation and try to
	// get it from environment variables set by pod.
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return fmt.Errorf("initAPIHost: not running in cluster: KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT environment variables not set")
	}

	c.apiHost = net.JoinHostPort(host, port)
	return nil
}

func (c *client) httpTransportWithTLS() (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}

	if c.cfg.GetTlsConfig() != nil {
		if err := tlsconfig.UpdateTLSConfig(transport.TLSClientConfig, c.cfg.GetTlsConfig()); err != nil {
			return nil, err
		}
		return transport, nil
	}

	// If TLS config is not provided, assume in-cluster.
	certs, err := os.ReadFile(LocalCACert)
	if err != nil {
		return nil, fmt.Errorf("error while reading local ca.crt file (%s): %v", LocalCACert, err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certs)

	transport.TLSClientConfig.RootCAs = caCertPool

	return transport, nil
}

func newClientWithoutToken(cfg *configpb.ProviderConfig, l *logger.Logger) (*client, error) {
	c := &client{
		cfg: cfg,
		l:   l,
	}

	transport, err := c.httpTransportWithTLS()
	if err != nil {
		return nil, err
	}

	c.httpC = &http.Client{
		Transport: transport,
	}

	if err := c.initAPIHost(); err != nil {
		return nil, err
	}

	return c, nil
}

func newClient(cfg *configpb.ProviderConfig, l *logger.Logger) (*client, error) {
	c, err := newClientWithoutToken(cfg, l)
	if err != nil {
		return nil, err
	}

	ts, err := oauth.K8STokenSource(l)
	if err != nil {
		return nil, fmt.Errorf("error while creating token source from k8s token file: %v", err)
	}

	c.httpC.Transport = &oauth2.Transport{
		Source: ts,
		Base:   c.httpC.Transport,
	}
	return c, nil
}
