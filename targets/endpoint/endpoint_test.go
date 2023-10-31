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

package endpoint

import (
	"fmt"
	"net"
	"testing"
	"time"

	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestEndpointsFromNames(t *testing.T) {
	names := []string{"targetA", "targetB", "targetC"}
	endpoints := EndpointsFromNames(names)

	for i := range names {
		ep := endpoints[i]

		if ep.Name != names[i] {
			t.Errorf("Endpoint.Name=%s, want=%s", ep.Name, names[i])
		}
		if ep.Port != 0 {
			t.Errorf("Endpoint.Port=%d, want=0", ep.Port)
		}
		if len(ep.Labels) != 0 {
			t.Errorf("Endpoint.Labels=%v, want={}", ep.Labels)
		}
	}
}

func TestKey(t *testing.T) {
	for _, test := range []struct {
		name   string
		port   int
		labels map[string]string
		ip     net.IP
		key    string
	}{
		{
			name: "t1",
			port: 80,
			ip:   net.ParseIP("10.0.0.1"),
			key:  "t1_10.0.0.1_80",
		},
		{
			name:   "t1",
			port:   80,
			ip:     net.ParseIP("1234:5678::72"),
			labels: map[string]string{"app": "cloudprober", "dc": "xx"},
			key:    "t1_1234:5678::72_80_app:cloudprober_dc:xx",
		},
		{
			name:   "t1",
			port:   80,
			labels: map[string]string{"dc": "xx", "app": "cloudprober"},
			key:    "t1__80_app:cloudprober_dc:xx",
		},
	} {
		ep := Endpoint{
			Name:   test.name,
			Port:   test.port,
			IP:     test.ip,
			Labels: test.labels,
		}
		t.Run(fmt.Sprintf("%v", ep), func(t *testing.T) {
			key := ep.Key()
			if key != test.key {
				t.Errorf("Got key: %s, want: %s", key, test.key)
			}
		})
	}
}

func TestEndpointDst(t *testing.T) {
	tests := []struct {
		ep   Endpoint
		want string
	}{
		{
			ep: Endpoint{
				Name: "ep-1",
			},
			want: "ep-1",
		},
		{
			ep: Endpoint{
				Name: "ep-2",
				Port: 9313,
			},
			want: "ep-2:9313",
		},
	}
	for _, tt := range tests {
		t.Run(tt.ep.Key(), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ep.Dst(), "destination")
		})
	}
}

func TestParseURL(t *testing.T) {
	type parts struct {
		scheme, host, path string
		port               int
	}
	tests := []struct {
		name      string
		s         string
		wantParts parts
		wantErr   bool
	}{
		{
			s:         "http://host:8080/path",
			wantParts: parts{scheme: "http", host: "host", path: "/path", port: 8080},
		},
		{
			s:         "https://host:8080",
			wantParts: parts{scheme: "https", host: "host", path: "/", port: 8080},
		},
		{
			s:         "http://host/path?query=1",
			wantParts: parts{scheme: "http", host: "host", path: "/path?query=1", port: 0},
		},
		{
			s:         "http://[abcf::0000]/path?query=1",
			wantParts: parts{scheme: "http", host: "abcf::0000", path: "/path?query=1", port: 0},
		},
		{
			s:         "http://[abcf::0000]:8080/path?query=1",
			wantParts: parts{scheme: "http", host: "abcf::0000", path: "/path?query=1", port: 8080},
		},
		{
			s:         "http://[abcf::0000]:8080/path?query=1",
			wantParts: parts{scheme: "http", host: "abcf::0000", path: "/path?query=1", port: 8080},
		},
		{
			s:       "http://[abcf::0000/path?query=1",
			wantErr: true,
		},
		{
			s:       "http://[abcf::0000]:asd/path?query=1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			gotScheme, gotHost, gotPath, gotPort, err := parseURL(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantParts.scheme, gotScheme, "scheme")
			assert.Equal(t, tt.wantParts.host, gotHost, "host")
			assert.Equal(t, tt.wantParts.path, gotPath, "path")
			assert.Equal(t, tt.wantParts.port, gotPort, "port")
		})
	}
}

func TestFromProtoMessage(t *testing.T) {
	tests := []struct {
		name        string
		endpointspb []*targetspb.Endpoint
		want        []Endpoint
		wantErr     bool
	}{
		{
			name: "static endpoints",
			endpointspb: []*targetspb.Endpoint{
				{
					Name: proto.String("host1_url1"),
					Url:  proto.String("http://host1:8080/url1"),
				},
				{
					Name:   proto.String("host2"),
					Url:    proto.String("https://host2.com"),
					Labels: map[string]string{"app": "frontend-cloudprober"},
				},
			},
			want: []Endpoint{
				{
					Name: "host1_url1",
					Labels: map[string]string{
						"__cp_scheme__": "http",
						"__cp_host__":   "host1",
						"__cp_path__":   "/url1",
					},
					Port: 8080,
				},
				{
					Name: "host2",
					Labels: map[string]string{
						"app":           "frontend-cloudprober",
						"__cp_scheme__": "https",
						"__cp_host__":   "host2.com",
						"__cp_path__":   "/",
					},
				},
			},
		},
		{
			name: "same keys error",
			endpointspb: []*targetspb.Endpoint{
				{
					Name: proto.String("host1_url1"),
					Url:  proto.String("http://host1:8080/url1"),
				},
				{
					Name: proto.String("host1_url1"),
					Url:  proto.String("http://host1:8080/url1"),
				},
			},
			wantErr: true,
		},
		{
			name: "different keys",
			endpointspb: []*targetspb.Endpoint{
				{
					Name: proto.String("host1"),
					Url:  proto.String("http://host1/url1"),
				},
				{
					Name: proto.String("host1"),
					Url:  proto.String("http://host1/url2"),
				},
			},
			want: []Endpoint{
				{
					Name: "host1",
					Labels: map[string]string{
						"__cp_scheme__": "http",
						"__cp_host__":   "host1",
						"__cp_path__":   "/url1",
					},
				},
				{
					Name: "host1",
					Labels: map[string]string{
						"__cp_scheme__": "http",
						"__cp_host__":   "host1",
						"__cp_path__":   "/url2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromProtoMessage(tt.endpointspb)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromProtoMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i := range got {
				got[i].LastUpdated = time.Time{}
			}
			assert.Equal(t, tt.want, got, "endpoints")
		})
	}
}

type testResolver struct {
	data map[string]net.IP
}

func (tr *testResolver) Resolve(name string, ipVersion int) (net.IP, error) {
	if tr.data == nil {
		tr.data = make(map[string]net.IP)
	}
	if tr.data[name] == nil {
		return nil, fmt.Errorf("no such host: %s", name)
	}
	return tr.data[name], nil
}

func TestEndpointResolve(t *testing.T) {
	res := &testResolver{
		data: map[string]net.IP{
			"host1": net.ParseIP("10.10.3.4"),
			"host2": net.ParseIP("2001:db8::1"),
		},
	}

	tests := []struct {
		name      string
		ep        Endpoint
		ipVersion int
		opts      []ResolverOption
		wantIP    string
		wantErr   bool
	}{
		{
			name:   "contains valid ip",
			ep:     Endpoint{Name: "host0", IP: net.ParseIP("10.1.1.1")},
			wantIP: "10.1.1.1",
		},
		{
			name:      "no ipv6",
			ep:        Endpoint{Name: "host0", IP: net.ParseIP("10.1.1.1")},
			ipVersion: 6,
			wantErr:   true,
		},
		{
			name:   "use_resolver",
			ep:     Endpoint{Name: "host1"},
			wantIP: "10.10.3.4",
		},
		{
			name:      "name_override",
			ep:        Endpoint{Name: "host0"},
			opts:      []ResolverOption{WithNameOverride("host2")},
			ipVersion: 6,
			wantIP:    "2001:db8::1",
		},
		{
			name:    "no host",
			ep:      Endpoint{Name: "host0"},
			opts:    []ResolverOption{WithNameOverride("host3")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ep.Resolve(tt.ipVersion, res, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Endpoint.Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantIP, got.String(), "resolved IP")
		})
	}
}
