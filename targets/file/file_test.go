// Copyright 2021 The Cloudprober Authors.
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

package file

import (
	"testing"

	"github.com/cloudprober/cloudprober/internal/rds/file/testdata"
	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	configpb "github.com/cloudprober/cloudprober/targets/file/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var (
	testExpectedEndpoints = testdata.ExpectedEndpoints()
	testExpectedIP        = testdata.ExpectedIPs()
)

func TestListEndpointsWithFilter(t *testing.T) {
	for _, test := range []struct {
		desc          string
		f             []*rdspb.Filter
		wantEndpoints []endpoint.Endpoint
	}{
		{
			desc:          "no_filter",
			wantEndpoints: testExpectedEndpoints,
		},
		{
			desc: "with_filter",
			f: []*rdspb.Filter{{
				Key:   proto.String("labels.cluster"),
				Value: proto.String("xx"),
			}},
			wantEndpoints: testExpectedEndpoints[:2],
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ft, err := New(&configpb.TargetsConf{
				FilePath: proto.String("../../internal/rds/file/testdata/targets.json"),
				Filter:   test.f,
			}, nil, nil)

			if err != nil {
				t.Fatalf("Unexpected error while parsing textpb: %v", err)
			}

			got := ft.ListEndpoints()

			if len(got) != len(test.wantEndpoints) {
				t.Fatalf("Got endpoints: %d, expected: %d", len(got), len(test.wantEndpoints))
			}

			for i := range test.wantEndpoints {
				want := test.wantEndpoints[i]

				assert.Equal(t, got[i].Name, want.Name)
				assert.Equal(t, got[i].Port, want.Port)
				assert.Equal(t, got[i].Labels, want.Labels)
				assert.Equal(t, got[i].IP.String(), want.IP.String())
			}
		})
	}

}
