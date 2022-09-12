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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/rds/kubernetes/proto"
)

// Variables defined by Kubernetes spec to find out local CA cert and token.
var (
	LocalCACert    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	LocalTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// client encapsulates an in-cluster kubeapi client.
type client struct {
	cfg           *configpb.ProviderConfig
	httpC         *http.Client
	apiHost       string
	bearer        string
	labelSelector string
	l             *logger.Logger
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

	if c.labelSelector != "" {
		values := req.URL.Query()
		values.Add("labelSelector", c.labelSelector)
		req.URL.RawQuery = values.Encode()
	}

	return req, nil
}

func (c *client) getURL(url string) ([]byte, error) {
	req, err := c.httpRequest(url)
	if err != nil {
		return nil, err
	}

	c.l.Infof("kubernetes.client: getting URL: %s", req.URL.String())
	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP response status code: %d, status: %s", resp.StatusCode, resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
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

func (c *client) initHTTPClient() error {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}

	if c.cfg.GetTlsConfig() != nil {
		if err := tlsconfig.UpdateTLSConfig(transport.TLSClientConfig, c.cfg.GetTlsConfig()); err != nil {
			return err
		}
		c.httpC = &http.Client{Transport: transport}
		return nil
	}

	// If TLS config is not provided, assume in-cluster.
	certs, err := ioutil.ReadFile(LocalCACert)
	if err != nil {
		return fmt.Errorf("error while reading local ca.crt file (%s): %v", LocalCACert, err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certs)

	transport.TLSClientConfig.RootCAs = caCertPool

	c.httpC = &http.Client{Transport: transport}

	// Read OAuth token from local file.
	token, err := ioutil.ReadFile(LocalTokenFile)
	if err != nil {
		return fmt.Errorf("error while reading in-cluster local token file (%s): %v", LocalTokenFile, err)
	}
	c.bearer = "Bearer " + string(token)

	return nil
}

func (c *client) parseLabelSelector() error {
	if len(c.cfg.GetLabelSelector()) == 0 {
		return nil
	}
	var labels []string

	for _, ls := range c.cfg.GetLabelSelector() {
		op := ls.GetOp()

		if op == "" {
			op = "="
		} else if op != "=" && op != "!=" {
			return fmt.Errorf("label selector operator (%s) can only be \"=\" or \"!=\"", op)
		}

		labels = append(labels, fmt.Sprintf("%s%s%s", ls.GetKey(), op, ls.GetValue()))
	}
	c.labelSelector = strings.Join(labels, ",")
	return nil
}

func newClient(cfg *configpb.ProviderConfig, l *logger.Logger) (*client, error) {
	c := &client{
		cfg: cfg,
		l:   l,
	}

	if err := c.initHTTPClient(); err != nil {
		return nil, err
	}

	if err := c.initAPIHost(); err != nil {
		return nil, err
	}

	if err := c.parseLabelSelector(); err != nil {
		return nil, err
	}

	return c, nil
}
