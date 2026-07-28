// Copyright 2026 The Cloudprober Authors.
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

package dns

import (
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/validators/dns/proto"
	"github.com/cloudprober/cloudprober/logger"
	mdns "github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
)

func TestInit(t *testing.T) {
	for _, test := range []struct {
		desc    string
		config  *configpb.Validator
		wantErr bool
	}{
		{"authoritative", &configpb.Validator{Authoritative: proto.Bool(true)}, false},
		{"no_criteria", &configpb.Validator{}, true},
		{"nil_config", nil, true},
	} {
		t.Run(test.desc, func(t *testing.T) {
			v := &Validator{}
			if err := v.Init(test.config); (err != nil) != test.wantErr {
				t.Errorf("got err: %v, want err: %v", err, test.wantErr)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		desc          string
		authoritative bool
		resp          interface{}
		wantResult    bool
		wantErr       bool
	}{
		{
			desc:          "aa_required_and_set",
			authoritative: true,
			resp:          &mdns.Msg{MsgHdr: mdns.MsgHdr{Authoritative: true}},
			wantResult:    true,
		},
		{
			desc:          "aa_required_but_not_set",
			authoritative: true,
			resp:          &mdns.Msg{},
			wantResult:    false,
		},
		{
			desc:          "aa_not_wanted_but_set",
			authoritative: false,
			resp:          &mdns.Msg{MsgHdr: mdns.MsgHdr{Authoritative: true}},
			wantResult:    false,
		},
		{
			desc:          "aa_not_wanted_and_not_set",
			authoritative: false,
			resp:          &mdns.Msg{},
			wantResult:    true,
		},
		{
			desc:          "bad_input_type",
			authoritative: true,
			resp:          "not a dns message",
			wantErr:       true,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			v := &Validator{}
			if err := v.Init(&configpb.Validator{Authoritative: proto.Bool(test.authoritative)}); err != nil {
				t.Fatalf("Error initializing validator: %v", err)
			}

			result, err := v.Validate(test.resp, nil, &logger.Logger{})
			if (err != nil) != test.wantErr {
				t.Errorf("got err: %v, want err: %v", err, test.wantErr)
			}
			if err != nil {
				return
			}
			if result != test.wantResult {
				t.Errorf("got result: %v, want: %v", result, test.wantResult)
			}
		})
	}
}
