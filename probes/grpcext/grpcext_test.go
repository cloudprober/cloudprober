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

package grpcext

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	validatorpb "github.com/cloudprober/cloudprober/internal/validators/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// fakeSidecar is a scripted Prober server.
type fakeSidecar struct {
	configpb.UnimplementedProberServer

	respFn func(req *configpb.ProbeRequest) *configpb.ProbeResponse
	// block makes Probe hang until the RPC context expires, simulating a
	// probe that consumes the full deadline (e.g. a hung target).
	block bool
	// validateErr is returned by ValidateConfig (nil => OK).
	validateErr error

	mu   sync.Mutex
	reqs []*configpb.ProbeRequest
}

func (f *fakeSidecar) Probe(ctx context.Context, req *configpb.ProbeRequest) (*configpb.ProbeResponse, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	f.mu.Unlock()
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.respFn(req), nil
}

func (f *fakeSidecar) ValidateConfig(ctx context.Context, req *configpb.ValidateConfigRequest) (*configpb.ValidateConfigResponse, error) {
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return &configpb.ValidateConfigResponse{}, nil
}

func (f *fakeSidecar) requests() []*configpb.ProbeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*configpb.ProbeRequest{}, f.reqs...)
}

// startFakeSidecar runs f on a unix socket and returns the server address.
func startFakeSidecar(t *testing.T, f *fakeSidecar) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "sidecar.sock")
	lis, err := net.Listen("unix", sock)
	require.NoError(t, err)
	srv := grpc.NewServer()
	configpb.RegisterProberServer(srv, f)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return "unix://" + sock
}

func newOpts(t *testing.T, server string) *options.Options {
	t.Helper()
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("testhost")
	opts.Timeout = 5 * time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{
		Server:    proto.String(server),
		ProbeType: proto.String("testtype"),
		Config:    proto.String(`{"key": "val"}`),
	}
	return opts
}

func initProbe(t *testing.T, opts *options.Options) *Probe {
	t.Helper()
	p := &Probe{}
	require.NoError(t, p.Init("test_probe", opts))
	return p
}

func runProbeOnce(t *testing.T, p *Probe, runReq *sched.RunProbeForTargetRequest) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.runProbe(ctx, runReq)
}

func newRunReq() *sched.RunProbeForTargetRequest {
	return &sched.RunProbeForTargetRequest{
		Target:  endpoint.Endpoint{Name: "testhost", Port: 8080},
		LastRun: &sched.LastRunResult{},
	}
}

func TestProbeSuccess(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{
				Success: true,
				Latency: durationpb.New(5 * time.Millisecond),
				Payload: []string{"rows_scanned 12"},
			}
		},
	}
	opts := newOpts(t, startFakeSidecar(t, f))
	p := initProbe(t, opts)

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)

	assert.True(t, runReq.LastRun.Success)
	assert.Equal(t, 5*time.Millisecond, runReq.LastRun.Latency)

	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(1), result.success)
	assert.Equal(t, int64(0), result.internalErrors)

	// Standard EM + payload metrics EM.
	ems := result.Metrics(time.Now(), 1, opts)
	require.Equal(t, 2, len(ems))
	assert.Equal(t, "external_grpc", ems[0].Label("ptype"))
	assert.Equal(t, "12.000", ems[1].Metric("rows_scanned").String())

	// Verify what the sidecar saw.
	reqs := f.requests()
	require.Equal(t, 1, len(reqs))
	assert.Equal(t, "testtype", reqs[0].GetProbeType())
	assert.Equal(t, "testhost", reqs[0].GetTarget().GetName())
	assert.Equal(t, int32(8080), reqs[0].GetTarget().GetPort())
	assert.Equal(t, `{"key": "val"}`, string(reqs[0].GetConfig()))
}

func TestProbeTargetFailure(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{Success: false, Error: "connection refused"}
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)

	assert.False(t, runReq.LastRun.Success)
	assert.ErrorContains(t, runReq.LastRun.Error, "connection refused")

	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(0), result.success)
	assert.Equal(t, int64(0), result.internalErrors)
}

func TestProbeInternalError(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{InternalError: true, Error: "db driver crashed"}
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)

	assert.False(t, runReq.LastRun.Success)
	assert.ErrorContains(t, runReq.LastRun.Error, "db driver crashed")

	// Internal errors count as failed runs (total moves, success doesn't) so
	// alerting keeps working; internal_errors additionally records the cause.
	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(0), result.success)
	assert.Equal(t, int64(1), result.internalErrors)
}

func TestProbeSidecarDown(t *testing.T) {
	// Point the probe at a socket that nothing listens on.
	sock := filepath.Join(t.TempDir(), "nobody-home.sock")
	p := initProbe(t, newOpts(t, "unix://"+sock))

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)

	assert.False(t, runReq.LastRun.Success)
	assert.Error(t, runReq.LastRun.Error)

	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(0), result.success)
	assert.Equal(t, int64(1), result.internalErrors)
}

func TestProbeDeadline(t *testing.T) {
	// A probe that consumes the whole budget is a hung target, not a sidecar
	// failure: it must count as a plain probe failure, not internal error.
	f := &fakeSidecar{block: true}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	runReq := newRunReq()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	p.runProbe(ctx, runReq)

	assert.False(t, runReq.LastRun.Success)
	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(0), result.success)
	assert.Equal(t, int64(0), result.internalErrors)
}

func TestProbeValidators(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{Success: true, Payload: []string{"rows_scanned 12"}}
		},
	}
	addr := startFakeSidecar(t, f)

	for _, tc := range []struct {
		regex       string
		wantSuccess int64
	}{
		{"rows_scanned 1[0-9]", 1},
		{"rows_scanned 9[0-9]", 0},
	} {
		opts := newOpts(t, addr)
		var err error
		opts.Validators, err = validators.Init([]*validatorpb.Validator{{
			Name: "rows",
			Type: &validatorpb.Validator_Regex{Regex: tc.regex},
		}})
		require.NoError(t, err)
		p := initProbe(t, opts)

		runReq := newRunReq()
		runProbeOnce(t, p, runReq)

		result := runReq.Result.(*probeResult)
		assert.Equal(t, int64(1), result.total, tc.regex)
		assert.Equal(t, tc.wantSuccess, result.success, tc.regex)

		ems := result.Metrics(time.Now(), 1, opts)
		assert.Equal(t, int64(1-tc.wantSuccess), ems[0].Metric("validation_failure").(*metrics.Map[int64]).GetKey("rows"), tc.regex)
	}
}

func TestProbeStateHandle(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			resp := &configpb.ProbeResponse{Success: true, StateHandle: req.GetStateHandle()}
			if len(req.GetStateHandle()) == 0 {
				resp.StateHandle = []byte("session-1")
			}
			return resp
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)
	runProbeOnce(t, p, runReq)

	// Handle minted on the first response is stored on TargetState and
	// echoed in the second request.
	assert.Equal(t, []byte("session-1"), runReq.TargetState)
	reqs := f.requests()
	require.Equal(t, 2, len(reqs))
	assert.Empty(t, reqs[0].GetStateHandle())
	assert.Equal(t, []byte("session-1"), reqs[1].GetStateHandle())
}

func TestRunOnceHandleReuse(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			resp := &configpb.ProbeResponse{Success: true, StateHandle: req.GetStateHandle()}
			if len(req.GetStateHandle()) == 0 {
				resp.StateHandle = []byte("session-1")
			}
			return resp
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	// RunOnce-style: every invocation gets a throwaway runReq with nil
	// TargetState. The probe-level handle cache must still reuse the
	// sidecar session instead of minting one per call.
	runProbeOnce(t, p, newRunReq())
	runProbeOnce(t, p, newRunReq())

	reqs := f.requests()
	require.Equal(t, 2, len(reqs))
	assert.Empty(t, reqs[0].GetStateHandle())
	assert.Equal(t, []byte("session-1"), reqs[1].GetStateHandle())
}

func TestInitErrors(t *testing.T) {
	opts := newOpts(t, "")
	opts.ProbeConf.(*configpb.ProbeConf).Server = nil
	assert.ErrorContains(t, (&Probe{}).Init("p", opts), "server")

	opts = newOpts(t, "localhost:1234")
	opts.ProbeConf.(*configpb.ProbeConf).ProbeType = nil
	assert.ErrorContains(t, (&Probe{}).Init("p", opts), "probe_type")
}

func TestInitValidatesConfigWithSidecar(t *testing.T) {
	// Sidecar rejects the config (unknown probe type / bad config) => Init
	// must fail fast.
	f := &fakeSidecar{validateErr: status.Errorf(codes.NotFound, "unknown probe type")}
	opts := newOpts(t, startFakeSidecar(t, f))
	assert.ErrorContains(t, (&Probe{}).Init("p", opts), "rejected config")

	// Sidecar accepts => Init succeeds.
	f.validateErr = nil
	require.NoError(t, (&Probe{}).Init("p", opts))

	// Unreachable sidecar is not an Init error: it may just not be up yet.
	sock := filepath.Join(t.TempDir(), "nobody-home.sock")
	require.NoError(t, (&Probe{}).Init("p", newOpts(t, "unix://"+sock)))
}
