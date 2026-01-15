// Copyright 2017-2024 The Cloudprober Authors.
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
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	validatorpb "github.com/cloudprober/cloudprober/internal/validators/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/dns/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
)

// If question contains a bad domain or type, DNS query response status should
// contain an error.
const (
	questionBadDomain    = "nosuchname"
	questionBadType      = configpb.QueryType_CAA
	answerContent        = " 3600 IN A 192.168.0.1"
	answerMatchPattern   = "3600"
	answerNoMatchPattern = "NAA"
)

var (
	globalLog = logger.Logger{}
)

type mockClient struct{}

// Exchange implementation that returns an error status if the query is for
// questionBad[Domain|Type]. This allows us to check if query parameters are
// populated correctly.
func (*mockClient) ExchangeContext(ctx context.Context, in *dns.Msg, fullTarget string) (*dns.Msg, time.Duration, error) {
	if fullTarget != "8.8.8.8:53" {
		return nil, 0, fmt.Errorf("unexpected target: %v", fullTarget)
	}
	out := &dns.Msg{}
	question := in.Question[0]
	if question.Name == questionBadDomain+"." || int(question.Qtype) == int(questionBadType) {
		out.Rcode = dns.RcodeNameError
	}
	answerStr := question.Name + answerContent
	a, err := dns.NewRR(answerStr)
	if err != nil {
		globalLog.Errorf("Error parsing answer \"%s\": %v", answerStr, err)
	} else {
		out.Answer = []dns.RR{a}
	}
	return out, time.Millisecond, nil
}
func (*mockClient) setTimeout(time.Duration)      {}
func (*mockClient) setSourceIP(net.IP)            {}
func (*mockClient) setDNSProto(configpb.DNSProto) {}

func runProbeAndVerify(t *testing.T, testName string, p *Probe, total, success int64) {
	p.client = new(mockClient)
	p.targets = p.opts.Targets.ListEndpoints()

	for _, target := range p.targets {
		runReq := &sched.RunProbeForTargetRequest{Target: target}
		p.runProbe(context.Background(), runReq)

		result := runReq.Result.(*probeRunResult)
		if result.total.Int64() != total || result.success.Int64() != success {
			t.Errorf("test(%s): result mismatch got (total, success) = (%d, %d), want (%d, %d)",
				testName, result.total.Int64(), result.success.Int64(), total, success)
		}
	}
}

func testVerifyProto(t *testing.T, testName string, p *Probe) {
	// DNSProto value is set in the non-mock client on Init and it is hidden by the mock on instantiation
	dnsClient := p.client.(*clientImpl)
	if dnsClient.Net != map[configpb.DNSProto]string{
		configpb.DNSProto_UDP:     "udp",
		configpb.DNSProto_TCP:     "tcp",
		configpb.DNSProto_TCP_TLS: "tcp-tls",
	}[p.c.GetDnsProto()] {
		t.Errorf("test(%s): mismatch between probe client DNSProto (%s) and config (%s)",
			testName, dnsClient.Net, p.c.GetDnsProto().String())
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		description string
		probeConf   *configpb.ProbeConf
		wantTotal   int64
		wantSuccess int64
		wantErr     bool
	}{
		{
			description: "basic",
			probeConf:   &configpb.ProbeConf{},
			wantTotal:   1,
			wantSuccess: 1,
		},
		{
			description: "error_in_config",
			probeConf: &configpb.ProbeConf{
				RequestsPerProbe:     proto.Int32(21),
				RequestsIntervalMsec: proto.Int32(100),
			},
			wantErr: true,
		},
		{
			description: "req_per_probe",
			probeConf: &configpb.ProbeConf{
				RequestsPerProbe: proto.Int32(2),
			},
			wantTotal:   2,
			wantSuccess: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			p := &Probe{}
			opts := &options.Options{
				Targets:   targets.StaticTargets("8.8.8.8"),
				Interval:  2 * time.Second,
				Timeout:   time.Second,
				ProbeConf: test.probeConf,
			}
			err := p.Init("dns_test", opts)
			if (err != nil) != test.wantErr {
				t.Errorf("got err: %v, want err: %v", err, test.wantErr)
			}
			if err != nil {
				return
			}
			runProbeAndVerify(t, "basic", p, test.wantTotal, test.wantSuccess)
		})
	}
}

type testTargets struct {
	Name string
	IP   net.IP
}

func (tt *testTargets) ListEndpoints() []endpoint.Endpoint {
	return []endpoint.Endpoint{{Name: tt.Name, IP: tt.IP}}
}

func (tt *testTargets) Resolve(string, int) (net.IP, error) {
	return nil, nil
}

func TestResolveFirst(t *testing.T) {
	p := &Probe{}
	opts := options.DefaultOptions()

	tt := &testTargets{Name: "foo", IP: net.ParseIP("8.8.8.8")}
	opts.Targets = tt
	opts.ProbeConf = &configpb.ProbeConf{ResolveFirst: proto.Bool(true)}
	if err := p.Init("dns_test_resolve_first", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		runProbeAndVerify(t, "resolve_first_success", p, 1, 1)
	})

	tt.IP = nil
	t.Run("error", func(t *testing.T) {
		runProbeAndVerify(t, "resolve_first_error", p, 1, 0)
	})
}

func TestProbeType(t *testing.T) {
	p := &Probe{}
	badType := questionBadType
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			QueryType: &badType,
		},
	}
	if err := p.Init("dns_probe_type_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbeAndVerify(t, "probetype", p, 1, 0)
}

func TestProbeProto(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:   targets.StaticTargets("8.8.8.8"),
		Interval:  2 * time.Second,
		Timeout:   time.Second,
		ProbeConf: &configpb.ProbeConf{},
	}
	if err := p.Init("dns_probe_proto_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}

	// expect success using defaults
	testVerifyProto(t, "probeprotoudpdefaulttest", p)

	// Testing explicit udp
	opts.ProbeConf = &configpb.ProbeConf{
		DnsProto: configpb.DNSProto_UDP.Enum(),
	}
	if err := p.Init("dns_probe_proto_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect success
	testVerifyProto(t, "probeprotoudptest", p)

	// Testing tcp
	opts.ProbeConf = &configpb.ProbeConf{
		DnsProto: configpb.DNSProto_TCP.Enum(),
	}
	if err := p.Init("dns_probe_proto_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect success
	testVerifyProto(t, "probeprototcptest", p)

	// Testing tcp-tls
	opts.ProbeConf = &configpb.ProbeConf{
		DnsProto: configpb.DNSProto_TCP_TLS.Enum(),
	}
	if err := p.Init("dns_probe_proto_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect success
	testVerifyProto(t, "probeprototcptlstest", p)
}

func TestBadName(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			ResolvedDomain: proto.String(questionBadDomain),
		},
	}
	if err := p.Init("dns_bad_domain_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbeAndVerify(t, "baddomain", p, 1, 0)
}

func TestAnswerCheck(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			MinAnswers: proto.Uint32(1),
		},
	}
	if err := p.Init("dns_probe_answer_check_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect success minAnswers == num answers returned == 1.
	runProbeAndVerify(t, "matchminanswers", p, 1, 1)

	opts.ProbeConf = &configpb.ProbeConf{
		MinAnswers: proto.Uint32(2),
	}
	if err := p.Init("dns_probe_answer_check_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect failure because only one answer returned and two wanted.
	runProbeAndVerify(t, "toofewanswers", p, 1, 0)
}

func TestValidator(t *testing.T) {
	p := &Probe{}
	for _, tst := range []struct {
		name      string
		pattern   string
		successCt int64
	}{
		{"match", answerMatchPattern, 1},
		{"nomatch", answerNoMatchPattern, 0},
	} {
		valPb := []*validatorpb.Validator{
			{
				Name: tst.name,
				Type: &validatorpb.Validator_Regex{Regex: tst.pattern},
			},
		}
		validator, err := validators.Init(valPb)
		if err != nil {
			t.Fatalf("Error initializing validator for pattern %v: %v", tst.pattern, err)
		}
		opts := &options.Options{
			Targets:    targets.StaticTargets("8.8.8.8"),
			Interval:   2 * time.Second,
			Timeout:    time.Second,
			ProbeConf:  &configpb.ProbeConf{},
			Validators: validator,
		}
		if err := p.Init("dns_probe_answer_"+tst.name, opts); err != nil {
			t.Fatalf("Error creating probe: %v", err)
		}
		runProbeAndVerify(t, tst.name, p, 1, tst.successCt)
	}
}
