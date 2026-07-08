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

package sidecar

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type counterConfig struct {
	FailAfter  int  `json:"fail_after"`
	Invalidate bool `json:"invalidate"`
}

// counterSession counts probe runs, letting tests observe session reuse.
type counterSession struct{ runs int }

var newSessions, closedSessions atomic.Int64

var counterProbe = ProbeType[counterConfig, *counterSession]{
	New: func(ctx context.Context, t Target, c counterConfig) (*counterSession, error) {
		newSessions.Add(1)
		return &counterSession{}, nil
	},
	Probe: func(ctx context.Context, t Target, c counterConfig, s *counterSession) *Result {
		s.runs++
		if c.FailAfter > 0 && s.runs > c.FailAfter {
			return Fail(fmt.Errorf("failing after %d runs", c.FailAfter))
		}
		res := OK(time.Millisecond).Metric("runs", s.runs, "target", t.Name)
		if c.Invalidate {
			res.InvalidateSession()
		}
		return res
	},
	Close: func(s *counterSession) { closedSessions.Add(1) },
}

var newFailProbe = ProbeType[struct{}, any]{
	New: func(ctx context.Context, t Target, c struct{}) (any, error) {
		return nil, errors.New("target db unreachable")
	},
	Probe: func(ctx context.Context, t Target, c struct{}, _ any) *Result {
		return OK(time.Millisecond)
	},
}

var panicProbe = ProbeType[struct{}, any]{
	Probe: func(ctx context.Context, t Target, c struct{}, _ any) *Result {
		panic("oops")
	},
}

var internalProbe = ProbeType[struct{}, any]{
	Probe: func(ctx context.Context, t Target, c struct{}, _ any) *Result {
		return Internal(errors.New("driver exploded"))
	},
}

// startServer runs Serve on a unix socket and returns a connected client.
func startServer(t *testing.T, opts ...Option) pb.ProberClient {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "test.sock")
	go func() {
		opts := append([]Option{
			Listen("unix://" + sock),
			IdleTTL(time.Minute),
			Register("counter", counterProbe),
			Register("panicky", panicProbe),
			Register("internal", internalProbe),
			Register("newfail", newFailProbe),
		}, opts...)
		// Serve only returns on error; the test process may be shutting
		// down, so log via stdlib log rather than t.Log.
		log.Println("Serve returned:", Serve(opts...))
	}()

	// Wait for the socket to show up.
	require.Eventually(t, func() bool {
		_, err := os.Stat(sock)
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)

	conn, err := grpc.NewClient("unix://"+sock, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return pb.NewProberClient(conn)
}

func probeReq(probeType, config string, handle []byte) *pb.ProbeRequest {
	return &pb.ProbeRequest{
		ProbeType:   probeType,
		Target:      &pb.Target{Name: "t1", Port: 443},
		Config:      []byte(config),
		StateHandle: handle,
	}
}

func TestServeSessionLifecycle(t *testing.T) {
	client := startServer(t)
	newSessionsBefore := newSessions.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First probe: no handle -> New is called, handle minted, runs=1.
	resp, err := client.Probe(ctx, probeReq("counter", "", nil))
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
	assert.NotEmpty(t, resp.GetStateHandle())
	assert.Equal(t, []string{`runs{target="t1"} 1`}, resp.GetPayload())

	// Second probe with the handle: session reused (runs=2, no new session).
	resp2, err := client.Probe(ctx, probeReq("counter", "", resp.GetStateHandle()))
	require.NoError(t, err)
	assert.Equal(t, []string{`runs{target="t1"} 2`}, resp2.GetPayload())
	assert.Equal(t, resp.GetStateHandle(), resp2.GetStateHandle())
	assert.Equal(t, newSessionsBefore+1, newSessions.Load())

	// Unknown/stale handle (e.g. sidecar restarted): transparently re-New.
	resp3, err := client.Probe(ctx, probeReq("counter", "", []byte("stale-handle")))
	require.NoError(t, err)
	assert.True(t, resp3.GetSuccess())
	assert.NotEqual(t, "stale-handle", string(resp3.GetStateHandle()))
	assert.Equal(t, newSessionsBefore+2, newSessions.Load())
}

func TestServeFailureAndConfig(t *testing.T) {
	client := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Config decoding: fail_after=1 makes the second run fail (target-level).
	resp, err := client.Probe(ctx, probeReq("counter", `{"fail_after": 1}`, nil))
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())

	resp, err = client.Probe(ctx, probeReq("counter", `{"fail_after": 1}`, resp.GetStateHandle()))
	require.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.False(t, resp.GetInternalError())
	assert.Contains(t, resp.GetError(), "failing after 1 runs")

	// Bad config -> internal error, not target failure.
	resp, err = client.Probe(ctx, probeReq("counter", `{not json`, nil))
	require.NoError(t, err)
	assert.True(t, resp.GetInternalError())
}

func TestServeInternalPaths(t *testing.T) {
	client := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tc := range []struct {
		probeType string
		wantErr   string
	}{
		{"nosuchtype", "unknown probe type"},
		{"panicky", "panicked"},
		{"internal", "driver exploded"},
	} {
		resp, err := client.Probe(ctx, probeReq(tc.probeType, "", nil))
		require.NoError(t, err, tc.probeType)
		assert.False(t, resp.GetSuccess(), tc.probeType)
		assert.True(t, resp.GetInternalError(), tc.probeType)
		assert.Contains(t, resp.GetError(), tc.wantErr, tc.probeType)
	}
}

func TestResultMetric(t *testing.T) {
	r := OK(time.Second).
		Metric("bytes", 4096).
		Metric("ratio", 0.75).
		Metric("bytes", 1024, "phase", "scan").
		Metric("note", "hi there", "k1", "v1", "k2", "v2").
		Metric("ok", true).
		Metric("lat", "dist:sum:899|count:221|lb:0,1|bc:34,54").
		Metric("odd", 1, "dangling")
	assert.Equal(t, []string{
		"bytes 4096",
		"ratio 0.75",
		`bytes{phase="scan"} 1024`,
		// Strings and bools are quoted so cloudprober's payload parser
		// accepts them; dist:/map: strings pass through raw; a dangling
		// label key gets an empty value.
		`note{k1="v1",k2="v2"} "hi there"`,
		`ok "true"`,
		"lat dist:sum:899|count:221|lb:0,1|bc:34,54",
		`odd{dangling=""} 1`,
	}, r.payload)
}

func TestServeNewError(t *testing.T) {
	client := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// New failing is a target-level failure (can't reach the target), NOT a
	// sidecar internal error.
	resp, err := client.Probe(ctx, probeReq("newfail", "", nil))
	require.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.False(t, resp.GetInternalError())
	assert.Contains(t, resp.GetError(), "target db unreachable")
}

func TestValidateConfig(t *testing.T) {
	client := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.ValidateConfig(ctx, &pb.ValidateConfigRequest{ProbeType: "counter", Config: []byte(`{"fail_after": 1}`)})
	assert.NoError(t, err)

	_, err = client.ValidateConfig(ctx, &pb.ValidateConfigRequest{ProbeType: "nosuchtype"})
	assert.Equal(t, codes.NotFound, status.Code(err))

	_, err = client.ValidateConfig(ctx, &pb.ValidateConfigRequest{ProbeType: "counter", Config: []byte(`{not json`)})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegisterErrors(t *testing.T) {
	err := Serve(
		Listen("unix:///tmp/unused.sock"),
		Register("dup", counterProbe),
		Register("dup", counterProbe),
	)
	assert.ErrorContains(t, err, `"dup" registered twice`)

	err = Serve(
		Listen("unix:///tmp/unused.sock"),
		Register("noprobe", ProbeType[struct{}, any]{}),
	)
	assert.ErrorContains(t, err, "Probe function is required")
}

func TestSessionInvalidate(t *testing.T) {
	client := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newBefore, closedBefore := newSessions.Load(), closedSessions.Load()

	// InvalidateSession => session is Closed after the run; the echoed
	// handle is stale, so the next probe builds a fresh session.
	resp, err := client.Probe(ctx, probeReq("counter", `{"invalidate": true}`, nil))
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
	assert.Equal(t, closedBefore+1, closedSessions.Load())

	resp2, err := client.Probe(ctx, probeReq("counter", `{"invalidate": true}`, resp.GetStateHandle()))
	require.NoError(t, err)
	// runs=1 again: fresh session, not the invalidated one.
	assert.Equal(t, []string{`runs{target="t1"} 1`}, resp2.GetPayload())
	assert.Equal(t, newBefore+2, newSessions.Load())
}

func TestIdleEviction(t *testing.T) {
	// Server with a very short TTL: the session created by one probe should
	// be swept and Closed shortly after (sweep interval floors at 1s).
	client := startServer(t, IdleTTL(100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	closedBefore := closedSessions.Load()
	_, err := client.Probe(ctx, probeReq("counter", "", nil))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return closedSessions.Load() > closedBefore
	}, 5*time.Second, 100*time.Millisecond, "idle session was never evicted/closed")
}
