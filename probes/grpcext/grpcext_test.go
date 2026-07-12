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
	"os"
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
	"google.golang.org/grpc/connectivity"
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

// tempSocketPath returns a path for a unix socket in a short-named temp
// directory (registered for cleanup), and the "unix:" address for it.
// Deliberately not t.TempDir(): it nests under the full (long) test name,
// which can push the socket path past macOS's ~104-byte sun_path limit.
func tempSocketPath(t *testing.T, name string) (path, addr string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "cpext")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path = filepath.Join(dir, name)
	// "unix:" (single colon, no slashes) round-trips through gRPC's target
	// parsing for both POSIX and Windows paths; "unix://" breaks on Windows
	// paths with a drive letter (see sidecar.Listen).
	return path, "unix:" + path
}

// startFakeSidecar runs f on a unix socket and returns the server address.
func startFakeSidecar(t *testing.T, f *fakeSidecar) string {
	t.Helper()
	sock, addr := tempSocketPath(t, "sidecar.sock")
	lis, err := net.Listen("unix", sock)
	require.NoError(t, err)
	srv := grpc.NewServer()
	configpb.RegisterProberServer(srv, f)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return addr
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
	p.runProbe(ctx, runReq, false)
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
	_, addr := tempSocketPath(t, "nobody-home.sock")
	p := initProbe(t, newOpts(t, addr))

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
	p.runProbe(ctx, runReq, false)

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

func TestProbeClearsStateHandleOnEmptyResponse(t *testing.T) {
	// Simulates the sidecar invalidating a session: the first response mints
	// a handle, the second explicitly clears it (empty state_handle), as
	// pkg/sidecar does after InvalidateSession.
	call := 0
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			call++
			if call == 1 {
				return &configpb.ProbeResponse{Success: true, StateHandle: []byte("session-1")}
			}
			return &configpb.ProbeResponse{Success: true}
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	runReq := newRunReq()
	runProbeOnce(t, p, runReq)
	assert.Equal(t, []byte("session-1"), runReq.TargetState)

	runProbeOnce(t, p, runReq)
	assert.Nil(t, runReq.TargetState)

	// Third request must not echo the discarded handle.
	runProbeOnce(t, p, runReq)
	reqs := f.requests()
	require.Equal(t, 3, len(reqs))
	assert.Equal(t, []byte("session-1"), reqs[1].GetStateHandle())
	assert.Empty(t, reqs[2].GetStateHandle())
}

func TestRunOnceIsOneShot(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{Success: true}
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	// Scheduled runs are not one-shot; RunOnce runs are, and carry no state
	// handle (the sidecar keeps no session for them).
	runProbeOnce(t, p, newRunReq())
	results := p.RunOnce(context.Background())
	require.Equal(t, 1, len(results))
	assert.True(t, results[0].Success)

	reqs := f.requests()
	require.Equal(t, 2, len(reqs))
	assert.False(t, reqs[0].GetOneShot())
	assert.True(t, reqs[1].GetOneShot())
	assert.Empty(t, reqs[1].GetStateHandle())
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
	_, addr := tempSocketPath(t, "nobody-home.sock")
	require.NoError(t, (&Probe{}).Init("p", newOpts(t, addr)))
}

func TestInitClosesConnOnRejectedConfig(t *testing.T) {
	// Init fails after the gRPC conn is already dialed (sidecar rejects the
	// config); the conn must not be leaked.
	f := &fakeSidecar{validateErr: status.Errorf(codes.NotFound, "unknown probe type")}
	opts := newOpts(t, startFakeSidecar(t, f))
	p := &Probe{}
	require.Error(t, p.Init("p", opts))
	require.NotNil(t, p.conn)
	assert.Equal(t, connectivity.Shutdown, p.conn.GetState())
}

func TestStartClosesConnOnExit(t *testing.T) {
	f := &fakeSidecar{
		respFn: func(req *configpb.ProbeRequest) *configpb.ProbeResponse {
			return &configpb.ProbeResponse{Success: true}
		},
	}
	p := initProbe(t, newOpts(t, startFakeSidecar(t, f)))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.Start(ctx, make(chan *metrics.EventMetrics, 10))
		close(done)
	}()

	require.Eventually(t, func() bool {
		return p.conn.GetState() == connectivity.Ready || p.conn.GetState() == connectivity.Idle
	}, 5*time.Second, 10*time.Millisecond, "probe never connected")

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
	assert.Equal(t, connectivity.Shutdown, p.conn.GetState())
}

// hungHandshakeListener accepts TCP connections but never writes anything —
// reachable at the network layer (no fast RST/ECONNREFUSED), but the gRPC
// handshake never completes. This is what makes DeadlineExceeded ambiguous:
// from the client's side, an RPC against this listener looks identical to
// "the handler accepted the RPC and ran long," except the transport never
// becomes Ready. A closed port, by contrast, fails fast with Unavailable
// (gRPC's default fail-fast behavior) and was never actually ambiguous.
func hungHandshakeListener(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { lis.Close() })
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
		}
	}()
	return lis.Addr().String()
}

func TestInternalErrorOnHungHandshake(t *testing.T) {
	p := initProbe(t, newOpts(t, hungHandshakeListener(t)))

	runReq := newRunReq()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	p.runProbe(ctx, runReq, false)

	assert.False(t, runReq.LastRun.Success)
	result := runReq.Result.(*probeResult)
	assert.Equal(t, int64(1), result.total)
	assert.Equal(t, int64(1), result.internalErrors)
}
