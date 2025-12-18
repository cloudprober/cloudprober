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

package http

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	oauthpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/internal/httpreq"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	probesconfigpb "github.com/cloudprober/cloudprober/probes/proto"

	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

func TestHostWithPort(t *testing.T) {
	for _, test := range []struct {
		host         string
		port         int
		wantHostPort string
	}{
		{
			host:         "target1.ns.cluster.local",
			wantHostPort: "target1.ns.cluster.local",
		},
		{
			host:         "target1.ns.cluster.local",
			port:         8080,
			wantHostPort: "target1.ns.cluster.local:8080",
		},
	} {
		t.Run(fmt.Sprintf("host:%s,port:%d", test.host, test.port), func(t *testing.T) {
			hostPort := hostWithPort(test.host, test.port)
			if hostPort != test.wantHostPort {
				t.Errorf("hostPort: %s, want: %s", hostPort, test.wantHostPort)
			}
		})
	}
}

func TestHandleIPv6(t *testing.T) {
	tests := map[string]string{
		"2600:2d00:4030:a47:c0a8:210d:0:0": "[2600:2d00:4030:a47:c0a8:210d:0:0]",
		"cloudprober.org":                  "cloudprober.org",
		"10.12.13.14":                      "10.12.13.14",
	}
	for in, want := range tests {
		assert.Equalf(t, want, handleIPv6(in), "handleIPv6(%s) is not as expected", in)
	}
}

func TestURLHostAndHeader(t *testing.T) {
	for _, test := range []struct {
		name            string
		fqdn            string
		probeHostHeader string
		port            int
		wantHostHeader  string
		wantURLHost     string
	}{
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "svc.target",
			port:            8080,
			wantHostHeader:  "svc.target",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "",
			port:            8080,
			wantHostHeader:  "target1.ns.cluster.local:8080",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "",
			port:            0,
			wantHostHeader:  "target1.ns.cluster.local",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "",
			probeHostHeader: "",
			port:            8080,
			wantHostHeader:  "target1:8080",
			wantURLHost:     "target1",
		},
		{
			name:            "target1",
			fqdn:            "",
			probeHostHeader: "",
			port:            0,
			wantHostHeader:  "target1",
			wantURLHost:     "target1",
		},
	} {
		t.Run("", func(t *testing.T) {
			target := endpoint.Endpoint{
				Name:   test.name,
				Labels: map[string]string{"fqdn": test.fqdn},
			}

			urlHost := hostForTarget(target)
			if urlHost != test.wantURLHost {
				t.Errorf("Got URL host: %s, want URL host: %s", urlHost, test.wantURLHost)
			}

			p := &Probe{
				c: &configpb.ProbeConf{
					Header: map[string]string{
						"Host":         test.probeHostHeader,
						"X-Probe-Name": "probe1",
					},
				},
			}

			req, _ := http.NewRequest("GET", "http://cloudprober.org", nil)
			p.setHeaders(req, urlHost, test.port)
			assert.Equal(t, test.wantHostHeader, req.Host, "host header mismatch")
			assert.Equal(t, "probe1", req.Header.Get("X-Probe-Name"), "probe name header mismatch")
		})
	}
}

func TestPathforTarget(t *testing.T) {
	allLabels := map[string]string{
		relURLLabel:   "/target-url",
		"__cp_path__": "target-cp-path", // "/" is added automatically.
	}

	// This is a map of wanted output to defined labels.
	tests := map[string][]string{
		"/target-url":     {relURLLabel, "__cp_path__"},
		"/target-cp-path": {"__cp_path__"},
	}

	for want, labelKeys := range tests {
		for _, probeURL := range []string{"", "/metrics"} {
			t.Run(fmt.Sprintf("want:%s,probeURL:%s", want, probeURL), func(t *testing.T) {
				labels := map[string]string{}
				for _, labelKey := range labelKeys {
					labels[labelKey] = allLabels[labelKey]
				}
				target := endpoint.Endpoint{
					Name:   "test-target",
					Labels: labels,
				}
				gotPath := pathForTarget(target, probeURL)
				if probeURL != "" {
					assert.Equal(t, probeURL, gotPath, "path is not set to non-empty probeURL")
				} else {
					assert.Equal(t, want, gotPath, "path is not set to expected value")
				}
			})
		}
	}
}

// Following tests are more comprehensive tests for request URL and host header.
type testData struct {
	desc         string
	targetName   string
	targetFQDN   string // Make target have this "fqdn" label.
	resolvedIP   string // IP that will be returned by the test resolve function.
	resolveFirst bool

	// Probe configuration parameters
	probeHost string

	wantError bool
	// Used by TestURLHostAndHeaderForTarget to verify URL host, and
	// TestRequestHostAndURL to build URL to verify.
	wantURLHost    string
	wantHostHeader string
}

func createRequestAndVerify(t *testing.T, td testData, probePort, targetPort, expectedPort int) {
	t.Helper()

	// Initialize proe
	p := &Probe{}
	opts := options.DefaultOptions()
	c := &configpb.ProbeConf{
		Port: proto.Int32(int32(probePort)),
		Header: map[string]string{
			"X-Probe-Name": "probe1",
		},
	}
	if td.resolveFirst {
		c.ResolveFirst = proto.Bool(true)
	}
	if td.probeHost != "" {
		c.Header["Host"] = td.probeHost
	}
	opts.ProbeConf = c
	opts.Targets = targets.StaticTargets(td.targetName)

	// Configure additional labels.
	for _, pair := range [][2]string{
		{"fixed_label", "fixed_value"},
		{"target_name", "@target.name@"},
		{"target_port", "@target.port@"},
		{"target_ip", "@target.ip@"},
	} {
		opts.AdditionalLabels = append(opts.AdditionalLabels, options.ParseAdditionalLabel(&probesconfigpb.AdditionalLabel{
			Key:   proto.String(pair[0]),
			Value: proto.String(pair[1]),
		}))
	}
	assert.NoError(t, p.Init("test", opts), "Error initializing probe")

	target := endpoint.Endpoint{
		Name: td.targetName,
		IP:   net.ParseIP(td.resolvedIP),
		Port: targetPort,
		Labels: map[string]string{
			"fqdn": td.targetFQDN,
		},
	}
	req, err := p.httpRequestForTarget(target)
	if td.wantError {
		assert.Error(t, err, "No error creating request", req)
	} else {
		assert.NoError(t, err, "Error creating request")
	}

	wantIPLabel := ""
	if target.IP != nil {
		wantIPLabel = target.IP.String()
	}
	wantLabels := [][2]string{
		{"fixed_label", "fixed_value"},
		{"target_name", td.targetName},
		{"target_port", strconv.Itoa(expectedPort)},
		{"target_ip", wantIPLabel},
	}
	var gotLabels [][2]string
	for _, al := range opts.AdditionalLabels {
		key, val := al.KeyValueForTarget(target)
		gotLabels = append(gotLabels, [2]string{key, val})
	}
	assert.Equal(t, wantLabels, gotLabels, "Additional labels mismatch")

	if !td.wantError {
		wantURL := fmt.Sprintf("http://%s", hostWithPort(td.wantURLHost, expectedPort))
		assert.Equal(t, wantURL, req.URL.String(), "URL mismatch")
		assert.Equal(t, td.wantHostHeader, req.Host, "host header mismatch")
		assert.Equal(t, "probe1", req.Header.Get("X-Probe-Name"), "probe name header mismatch")
	}
}

func testRequestHostAndURLWithDifferentPorts(t *testing.T, td testData) {
	t.Helper()

	for _, ports := range []struct {
		probePort    int
		targetPort   int
		expectedPort int
	}{
		{
			probePort:    0,
			targetPort:   0,
			expectedPort: 0,
		},
		{
			probePort:    8080,
			targetPort:   9313,
			expectedPort: 8080, // probe port wins
		},
		{
			probePort:    0,
			targetPort:   9313,
			expectedPort: 9313, // target port wins
		},
	} {
		t.Run(fmt.Sprintf("%s_probe_port_%d_endpoint_port_%d", td.desc, ports.probePort, ports.targetPort), func(t *testing.T) {
			td := td
			if td.probeHost == "" {
				td.wantHostHeader = hostWithPort(td.wantHostHeader, ports.expectedPort)
			}
			createRequestAndVerify(t, td, ports.probePort, ports.targetPort, ports.expectedPort)
		})
	}
}

func TestRequestHostAndURL(t *testing.T) {
	tests := []testData{
		// No resolve first
		{
			desc:           "no_resolve_first,no_probe_host_header",
			targetName:     "test-target.com",
			wantURLHost:    "test-target.com",
			wantHostHeader: "test-target.com",
		},
		{
			desc:           "no_resolve_first,fqdn,no_probe_host_header",
			targetName:     "test-target.com",
			targetFQDN:     "test.svc.cluster.local",
			wantURLHost:    "test.svc.cluster.local",
			wantHostHeader: "test.svc.cluster.local",
		},
		{
			desc:           "no_resolve_first,host_header",
			targetName:     "test-target.com",
			probeHost:      "test-host",
			wantURLHost:    "test-target.com",
			wantHostHeader: "test-host",
		},
		{
			desc:           "ipv6_literal_host,no_probe_host_header",
			targetName:     "2600:2d00:4030:a47:c0a8:210d:0:0", // IPv6 literal host
			wantURLHost:    "[2600:2d00:4030:a47:c0a8:210d:0:0]",
			wantHostHeader: "[2600:2d00:4030:a47:c0a8:210d:0:0]",
		},
		// Resolves first, as resolved IP is given.
		{
			desc:           "resolve_first,no_probe_host_header",
			targetName:     "localhost",
			resolvedIP:     "127.0.0.1", // Automatic resolve-first
			wantURLHost:    "127.0.0.1",
			wantHostHeader: "localhost",
		},
		{
			desc:           "resolve_first,ipv6,no_probe_host_header",
			targetName:     "localhost",
			resolvedIP:     "2600:2d00:4030:a47:c0a8:210d:0:0", // Resolved IP
			wantURLHost:    "[2600:2d00:4030:a47:c0a8:210d::]", // IPv6 literal host
			wantHostHeader: "localhost",
		},
		{
			desc:           "resolve_first,probe_host_header",
			targetName:     "localhost",
			probeHost:      "test-host",
			resolvedIP:     "127.0.0.1",
			wantURLHost:    "127.0.0.1",
			wantHostHeader: "test-host",
		},
		{
			desc:         "resolve_first,error_resolving",
			targetName:   "localhost_error_resolving",
			resolveFirst: true,
			wantError:    true,
		},
	}

	for _, td := range tests {
		t.Run(td.desc, func(t *testing.T) {
			testRequestHostAndURLWithDifferentPorts(t, td)
		})
	}
}

type fakeTokenSource struct {
	token string
}

func (fts *fakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: fts.token}, nil
}

func TestPrepareRequest(t *testing.T) {
	data := []string{}
	for i := 0; i < 500; i++ {
		data = append(data, fmt.Sprintf("data-%03d", i))
	}
	tests := []struct {
		name              string
		token             string
		token_type_format string
		data              []string
		wantIsCloned      bool
		wantNewBody       bool
	}{
		{
			name: "No token source, no body",
		},
		{
			name:         "No token source, small body",
			data:         data[0:20],
			wantIsCloned: true,
			wantNewBody:  true,
		},
		{
			name:         "No token source, large body",
			data:         data,
			wantIsCloned: true,
			wantNewBody:  true,
		},
		{
			name:         "token source, no body",
			token:        "test-token",
			wantIsCloned: true,
			wantNewBody:  false, // Only request is cloned.
		},
		{
			name:              "token source, custom token format",
			token:             "test-token",
			token_type_format: `Snowflake Token="%s"`,
			wantIsCloned:      true,
			wantNewBody:       false, // Only request is cloned.
		},
		{
			name:         "token source, small body",
			data:         data[0:20],
			token:        "test-token",
			wantIsCloned: true,
			wantNewBody:  true,
		},
		{
			name:         "token source, large body",
			data:         data,
			token:        "test-token",
			wantIsCloned: true,
			wantNewBody:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				requestBody: httpreq.NewRequestBody(tt.data...),
			}
			if tt.token != "" {
				p.oauthTS = &fakeTokenSource{token: tt.token}
			}
			if tt.token_type_format != "" {
				p.c = &configpb.ProbeConf{
					OauthConfig: &oauthpb.Config{
						TokenTypeFormat: &tt.token_type_format,
					},
				}
			}

			inReq, _ := httpreq.NewRequest("GET", "http://cloudprober.org", p.requestBody)
			got := p.prepareRequest(inReq)

			if tt.wantIsCloned != (inReq != got) {
				t.Errorf("wantIsCloned=%v, (inReq != got) is %v", tt.wantIsCloned, inReq != got)
			}

			if tt.wantNewBody != (inReq.Body != got.Body) {
				t.Errorf("wantNewBody=%v, (inReq.Body != got.Body) is %v", tt.wantNewBody, inReq.Body != got.Body)
			}

			if tt.token != "" {
				assert.Equal(t, fmt.Sprintf(p.c.GetOauthConfig().GetTokenTypeFormat(), tt.token), got.Header.Get("Authorization"), "Token mismatch")
			}

			if len(tt.data) != 0 {
				assert.NotNil(t, inReq.GetBody, "GetBody is nil")
				assert.NotNil(t, got.GetBody, "GetBody is nil")
			}
		})
	}
}

func TestRequestHasConfiguredHeaders(t *testing.T) {
	p := &Probe{}

	testHeaderName := "X-My-Test-Header"
	testHeaderValue := "foo"

	testHeadersName := "X-My-Other-Header"
	testHeadersValue := "bar"

	opts := &options.Options{
		Targets:  targets.StaticTargets("test.com"),
		Interval: 10 * time.Millisecond,
		ProbeConf: &configpb.ProbeConf{
			MaxRedirects: nil,
			Header:       map[string]string{testHeaderName: testHeaderValue},
			Headers:      []*configpb.ProbeConf_Header{{Name: &testHeadersName, Value: &testHeadersValue}},
		},
	}

	err := p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	target := endpoint.Endpoint{
		Name:   "header-test",
		Labels: map[string]string{"fqdn": "test.com"},
	}

	req, err := p.httpRequestForTarget(target)
	assert.NoError(t, err, "Error creating request")

	val, ok := req.Header[testHeaderName]
	assert.True(t, ok, "Configured header (via 'header' setting) is not present in target request")
	assert.Contains(t, val, testHeaderValue)

	val, ok = req.Header[testHeadersName]
	assert.True(t, ok, "Configured header (via 'headers' setting) is not present in target request")
	assert.Contains(t, val, testHeadersValue)
}

func TestResolveFirst(t *testing.T) {
	tests := []struct {
		name   string
		target endpoint.Endpoint
		conf   *configpb.ProbeConf
		want   bool
	}{
		{
			name:   "resolve_first_false",
			target: endpoint.Endpoint{Name: "cloudprober.org"},
			conf:   &configpb.ProbeConf{},
			want:   false,
		},
		{
			name:   "resolve_first_true",
			target: endpoint.Endpoint{Name: "cloudprober.org"},
			conf:   &configpb.ProbeConf{ResolveFirst: proto.Bool(true)},
			want:   true,
		},
		{
			name: "resolve_first_true_with_ip",
			target: endpoint.Endpoint{
				Name: "cloudprober.org",
				IP:   net.ParseIP("1.1.1.1"),
			},
			conf: &configpb.ProbeConf{},
			want: true,
		},
		{
			name: "resolve_first_explcitly_false",
			target: endpoint.Endpoint{
				Name: "cloudprober.org",
				IP:   net.ParseIP("1.1.1.1"),
			},
			conf: &configpb.ProbeConf{ResolveFirst: proto.Bool(false)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				c: tt.conf,
			}
			assert.Equal(t, tt.want, p.resolveFirst(tt.target), "resolveFirst is not as expected")
		})
	}
}
