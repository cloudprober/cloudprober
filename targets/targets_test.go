// Copyright 2017-2023 The Cloudprober Authors.
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

package targets

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	rdsclientpb "github.com/cloudprober/cloudprober/internal/rds/client/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	eppb "github.com/cloudprober/cloudprober/targets/endpoint/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	dnsRes "github.com/cloudprober/cloudprober/targets/resolver"
	testdatapb "github.com/cloudprober/cloudprober/targets/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type mockLister struct {
	list []endpoint.Endpoint
}

func (mldl *mockLister) ListEndpoints() []endpoint.Endpoint {
	return mldl.list
}

// TestList does not test the New function, and is specifically testing
// the implementation of targets directly
func TestList(t *testing.T) {
	staticHosts := []string{"hostA", "hostB", "hostC"}
	cpCom, mgCom := "cloudprober.org", "manugarg.com"
	listerEndpoint := []endpoint.Endpoint{
		{Name: cpCom, LastUpdated: time.Now()},
		{Name: mgCom, LastUpdated: time.Now()},
	}

	var tests = []struct {
		desc   string
		hosts  []string
		re     string
		ldList []endpoint.Endpoint
		want   []string
	}{
		{
			desc:   "hostB is lameduck",
			ldList: endpoint.EndpointsFromNames([]string{"hostB"}), // hostB is lameduck.
			want:   []string{"hostA", "hostC", cpCom, mgCom},
		},
		{
			desc: "all hosts no lameduck",
			re:   ".*",
			want: []string{"hostA", "hostB", "hostC", cpCom, mgCom},
		},
		{
			desc:   "only hosts starting with host and hostC is lameduck",
			re:     "host.*",
			ldList: endpoint.EndpointsFromNames([]string{"hostC"}), // hostC is lameduck.
			want:   []string{"hostA", "hostB"},
		},
		{
			desc:   "only hosts starting with host and hostC was lameducked before hostC was updated",
			re:     "host.*",
			ldList: []endpoint.Endpoint{{Name: "hostC", LastUpdated: time.Now().Add(-time.Hour)}}, // hostC is lameduck.
			want:   []string{"hostA", "hostB", "hostC"},
		},
		{
			desc: "empty as no hosts match the regex",
			re:   "empty.*",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			targetsDef := &targetspb.TargetsDef{
				Regex: proto.String(tt.re),
			}
			for _, ep := range staticHosts {
				targetsDef.Endpoint = append(targetsDef.Endpoint, &eppb.Endpoint{
					Name: proto.String(ep),
				})
			}

			bt, err := baseTargets(targetsDef, &mockLister{tt.ldList}, nil)
			assert.NoError(t, err, "Unexpected error building targets")

			bt.lister = &mockLister{listerEndpoint}

			assert.Equal(t, tt.want, endpoint.NamesFromEndpoints(bt.ListEndpoints()), "Unexpected targets")
		})
	}
}

func TestDummyTargets(t *testing.T) {
	targetsDef := &targetspb.TargetsDef{
		Type: &targetspb.TargetsDef_DummyTargets{
			DummyTargets: &targetspb.DummyTargets{},
		},
	}
	l := &logger.Logger{}
	tgts, err := New(targetsDef, nil, nil, nil, l)
	if err != nil {
		t.Fatalf("New(...) Unexpected errors %v", err)
	}
	got := endpoint.NamesFromEndpoints(tgts.ListEndpoints())
	want := []string{""}
	if !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("tgts.List() = %q, want %q", got, want)
	}
	ip, err := tgts.Resolve(got[0], 4)
	if err != nil {
		t.Errorf("tgts.Resolve(%q, 4) Unexpected errors %v", got[0], err)
	} else if !ip.IsUnspecified() {
		t.Errorf("tgts.Resolve(%q, 4) = %v is specified, expected unspecified", got[0], ip)
	}
	ip, err = tgts.Resolve(got[0], 6)
	if err != nil {
		t.Errorf("tgts.Resolve(%q, 6) Unexpected errors %v", got[0], err)
	} else if !ip.IsUnspecified() {
		t.Errorf("tgts.Resolve(%q, 6) = %v is specified, expected unspecified", got[0], ip)
	}
}

type testTargetsType struct {
	names []string
}

func (tgts *testTargetsType) List() []string {
	return tgts.names
}

func (tgts *testTargetsType) ListEndpoints() []endpoint.Endpoint {
	return endpoint.EndpointsFromNames(tgts.names)
}

func (tgts *testTargetsType) Resolve(name string, ipVer int) (net.IP, error) {
	return nil, errors.New("resolve not implemented")
}

func TestGetExtensionTargets(t *testing.T) {
	targetsDef := &targetspb.TargetsDef{}

	// This has the same effect as using the following in your config:
	// targets {
	//    [cloudprober.testdata.fancy_targets] {
	//      name: "fancy"
	//    }
	// }
	proto.SetExtension(targetsDef, testdatapb.E_FancyTargets, &testdatapb.FancyTargets{Name: proto.String("fancy")})
	tgts, err := New(targetsDef, nil, nil, nil, nil)
	if err == nil {
		t.Errorf("Expected error in building targets from extensions, got nil. targets: %v", tgts)
	}
	testTargets := []string{"a", "b"}
	RegisterTargetsType(200, func(conf interface{}, l *logger.Logger) (Targets, error) {
		return &testTargetsType{names: testTargets}, nil
	})
	tgts, err = New(targetsDef, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Got error in building targets from extensions: %v.", err)
	}
	tgtsList := endpoint.NamesFromEndpoints(tgts.ListEndpoints())
	if !reflect.DeepEqual(tgtsList, testTargets) {
		t.Errorf("Extended targets: tgts.List()=%v, expected=%v", tgtsList, testTargets)
	}
}

func TestSharedTargets(t *testing.T) {
	testHosts := []string{"host1", "host2"}

	// Create shared targets and re-use them.
	SetSharedTargets("shared_test_targets", StaticTargets(strings.Join(testHosts, ",")))

	var tgts [2]Targets

	for i := range tgts {
		targetsDef := &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_SharedTargets{SharedTargets: "shared_test_targets"},
		}

		var err error
		tgts[i], err = New(targetsDef, nil, nil, nil, nil)

		if err != nil {
			t.Errorf("got error while creating targets from shared targets: %v", err)
		}

		got := endpoint.NamesFromEndpoints(tgts[i].ListEndpoints())
		if !reflect.DeepEqual(got, testHosts) {
			t.Errorf("Unexpected targets: tgts.List()=%v, expected=%v", got, testHosts)
		}
	}
}

func TestRDSClientConf(t *testing.T) {
	provider := "test-provider"
	rPath := "test-rsources"

	var rows = []struct {
		desc       string
		localAddr  string
		globalAddr string
		provider   string
		wantErr    bool
		wantAddr   string
	}{
		{
			desc:     "Error as RDS server address is not initialized",
			provider: provider,
			wantErr:  true,
		},
		{
			desc:       "Pick global address",
			provider:   provider,
			globalAddr: "test-global-addr",
			wantAddr:   "test-global-addr",
		},
		{
			desc:       "Pick local address over global",
			provider:   provider,
			localAddr:  "test-local-addr",
			globalAddr: "test-global-addr",
			wantAddr:   "test-local-addr",
		},
		{
			desc:      "Error because no provider",
			provider:  "",
			localAddr: "test-local-addr",
			wantAddr:  "test-local-addr",
			wantErr:   true,
		},
	}

	for _, r := range rows {
		t.Run(r.desc, func(t *testing.T) {
			pb := &targetspb.RDSTargets{
				ResourcePath: proto.String(fmt.Sprintf("%s://%s", r.provider, rPath)),
			}
			if r.localAddr != "" {
				pb.RdsServerOptions = &rdsclientpb.ClientConf_ServerOptions{
					ServerAddress: proto.String(r.localAddr),
				}
			}

			globalOpts := &targetspb.GlobalTargetsOptions{}
			if r.globalAddr != "" {
				globalOpts.RdsServerOptions = &rdsclientpb.ClientConf_ServerOptions{
					ServerAddress: proto.String(r.globalAddr),
				}
			}

			_, cc, err := rdsClientConf(pb, globalOpts, nil)
			if (err != nil) != r.wantErr {
				t.Errorf("wantErr: %v, got err: %v", r.wantErr, err)
			}

			if err != nil {
				return
			}

			if cc.GetServerOptions().GetServerAddress() != r.wantAddr {
				t.Errorf("Got RDS server address: %s, wanted: %s", cc.GetServerOptions().GetServerAddress(), r.wantAddr)
			}
			if cc.GetRequest().GetProvider() != provider {
				t.Errorf("Got provider: %s, wanted: %s", cc.GetRequest().GetProvider(), provider)
			}
			if cc.GetRequest().GetResourcePath() != rPath {
				t.Errorf("Got resource path: %s, wanted: %s", cc.GetRequest().GetResourcePath(), rPath)
			}
		})
	}
}

func TestNew(t *testing.T) {
	var gotResolver dnsRes.Resolver
	tests := []struct {
		name       string
		targetsDef *targetspb.TargetsDef
		wantNames  []string
		wantErr    bool
	}{
		{
			name:       "no targets",
			targetsDef: &targetspb.TargetsDef{},
			wantErr:    true,
		},
		{
			name: "static endpoints",
			targetsDef: &targetspb.TargetsDef{
				Endpoint: []*eppb.Endpoint{
					{
						Name: proto.String("host1"),
					},
				},
			},
			wantNames: []string{"host1"},
		},
		{
			name: "static endpoints and static hosts",
			targetsDef: &targetspb.TargetsDef{
				Type: &targetspb.TargetsDef_HostNames{
					HostNames: "host2,host3",
				},
				Endpoint: []*eppb.Endpoint{
					{
						Name: proto.String("host1"),
					},
				},
			},
			wantNames: []string{"host1", "host2", "host3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.targetsDef, nil, nil, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			gotNames := endpoint.NamesFromEndpoints(got.ListEndpoints())
			if gotResolver == nil {
				gotResolver = got.(*targets).resolver
			} else {
				assert.Equal(t, gotResolver, got.(*targets).resolver, "Unexpected resolver")
			}
			assert.Equal(t, tt.wantNames, gotNames, "Unexpected targets")
		})
	}
}
