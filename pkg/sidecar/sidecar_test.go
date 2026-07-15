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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/testcerts"
	pb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func writeTestCerts(t *testing.T) (caFile, certFile, keyFile string) {
	t.Helper()
	caFile, certFile, keyFile, err := testcerts.Generate(t.TempDir())
	require.NoError(t, err)
	return caFile, certFile, keyFile
}

// freeTCPAddr returns a currently-free loopback TCP address. There's an
// inherent race between releasing it and Serve re-binding, but it's fine for
// a test — Serve needs a concrete address since it owns the listener.
func freeTCPAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return addr
}

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

var panicNewProbe = ProbeType[struct{}, any]{
	New: func(ctx context.Context, t Target, c struct{}) (any, error) {
		panic("setup oops")
	},
	Probe: func(ctx context.Context, t Target, c struct{}, _ any) *Result {
		return OK(time.Millisecond)
	},
}

var internalProbe = ProbeType[struct{}, any]{
	Probe: func(ctx context.Context, t Target, c struct{}, _ any) *Result {
		return Internal(errors.New("driver exploded"))
	},
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
	// paths with a drive letter (see Listen).
	return path, "unix:" + path
}

// startServer runs Serve on a unix socket and returns a connected client and
// the server's address (for tests that need to dial it a second time, e.g.
// for a health check).
func startServer(t *testing.T, opts ...Option) (pb.ProberClient, string) {
	t.Helper()
	sock, addr := tempSocketPath(t, "test.sock")
	go func() {
		opts := append([]Option{
			Listen(addr),
			IdleTTL(time.Minute),
			Register("counter", counterProbe),
			Register("panicky", panicProbe),
			Register("internal", internalProbe),
			Register("newfail", newFailProbe),
			Register("panickynew", panicNewProbe),
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

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return pb.NewProberClient(conn), addr
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
	client, _ := startServer(t)
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
	client, _ := startServer(t)
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
	client, _ := startServer(t)
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

func TestServeNewPanicIsContained(t *testing.T) {
	client, _ := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// A panic in New (session setup) must be contained the same way a panic
	// in Probe is — reported as an internal error, not crash the sidecar.
	resp, err := client.Probe(ctx, probeReq("panickynew", "", nil))
	require.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.True(t, resp.GetInternalError())
	assert.Contains(t, resp.GetError(), "setup oops")

	// The server is still alive and serving other probe types.
	resp, err = client.Probe(ctx, probeReq("counter", "", nil))
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
}

func TestServeHealthSERVING(t *testing.T) {
	_, addr := startServer(t)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, service := range []string{"", pb.Prober_ServiceDesc.ServiceName} {
		resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: service})
		require.NoError(t, err, "service %q", service)
		assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus(), "service %q", service)
	}
}

func TestCloseAllSessions(t *testing.T) {
	closedBefore := closedSessions.Load()
	s := &server{
		sessions: map[string]*session{
			"h1": {h: typedHandler[counterConfig, *counterSession]{pt: counterProbe}, val: &counterSession{}},
			"h2": {h: typedHandler[counterConfig, *counterSession]{pt: counterProbe}, val: &counterSession{}},
		},
	}

	s.closeAllSessions()

	assert.Equal(t, closedBefore+2, closedSessions.Load())
	assert.Empty(t, s.sessions)
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
	client, _ := startServer(t)
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
	client, _ := startServer(t)
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
	client, _ := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newBefore, closedBefore := newSessions.Load(), closedSessions.Load()

	// InvalidateSession => session is Closed after the run, and no handle is
	// echoed back, so the next probe builds a fresh session.
	resp, err := client.Probe(ctx, probeReq("counter", `{"invalidate": true}`, nil))
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
	assert.Empty(t, resp.GetStateHandle())
	assert.Equal(t, closedBefore+1, closedSessions.Load())

	resp2, err := client.Probe(ctx, probeReq("counter", `{"invalidate": true}`, resp.GetStateHandle()))
	require.NoError(t, err)
	// runs=1 again: fresh session, not the invalidated one.
	assert.Equal(t, []string{`runs{target="t1"} 1`}, resp2.GetPayload())
	assert.Equal(t, newBefore+2, newSessions.Load())
}

func TestOneShot(t *testing.T) {
	client, _ := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newBefore, closedBefore := newSessions.Load(), closedSessions.Load()

	// One-shot runs build a session, probe, and tear it down inline: no
	// handle is returned and nothing is cached.
	for i := 0; i < 2; i++ {
		req := probeReq("counter", "", nil)
		req.OneShot = true
		resp, err := client.Probe(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.GetSuccess())
		assert.Empty(t, resp.GetStateHandle())
		// Fresh session every time: runs is always 1.
		assert.Equal(t, []string{`runs{target="t1"} 1`}, resp.GetPayload())
	}
	assert.Equal(t, newBefore+2, newSessions.Load())
	assert.Equal(t, closedBefore+2, closedSessions.Load())
}

func TestIdleEviction(t *testing.T) {
	// Server with a very short TTL: the session created by one probe should
	// be swept and Closed shortly after (sweep interval floors at 1s).
	client, _ := startServer(t, IdleTTL(100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	closedBefore := closedSessions.Load()
	_, err := client.Probe(ctx, probeReq("counter", "", nil))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return closedSessions.Load() > closedBefore
	}, 5*time.Second, 100*time.Millisecond, "idle session was never evicted/closed")
}

func TestServeGracefulShutdown(t *testing.T) {
	sock, addr := tempSocketPath(t, "shutdown.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Serve(
			Listen(addr),
			ShutdownOn(ctx),
			Register("counter", counterProbe),
		)
	}()
	require.Eventually(t, func() bool {
		_, err := os.Stat(sock)
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewProberClient(conn)

	// Leave a live session behind, then cancel the shutdown context.
	closedBefore := closedSessions.Load()
	rpcCtx, rpcCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer rpcCancel()
	_, err = client.Probe(rpcCtx, probeReq("counter", "", nil))
	require.NoError(t, err)

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err, "Serve should return nil on graceful shutdown")
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after shutdown context was canceled")
	}
	// The session left behind must have been closed on shutdown.
	assert.Greater(t, closedSessions.Load(), closedBefore, "session was not closed on shutdown")
}

func TestServeMutualTLS(t *testing.T) {
	caFile, certFile, keyFile := writeTestCerts(t)
	addr := freeTCPAddr(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := Serve(
			Listen(addr),
			ShutdownOn(ctx),
			TLS(certFile, keyFile, caFile),
			Register("counter", counterProbe),
		)
		log.Println("Serve returned:", err)
	}()

	caPEM, err := os.ReadFile(caFile)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))
	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	dial := func(t *testing.T, tlsCfg *tls.Config) pb.ProberClient {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
		require.NoError(t, err)
		t.Cleanup(func() { conn.Close() })
		return pb.NewProberClient(conn)
	}

	// A client that presents the CA-signed cert succeeds. Retry until the
	// server is up (also confirms readiness for the negative case below).
	mtlsClient := dial(t, &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      pool,
		ServerName:   "localhost",
	})
	require.Eventually(t, func() bool {
		rpcCtx, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		_, err := mtlsClient.Probe(rpcCtx, probeReq("counter", "", nil))
		return err == nil
	}, 5*time.Second, 50*time.Millisecond, "mTLS probe never succeeded")

	// A client that omits its certificate is rejected by the handshake.
	noCertClient := dial(t, &tls.Config{RootCAs: pool, ServerName: "localhost"})
	rpcCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
	defer c()
	_, err = noCertClient.Probe(rpcCtx, probeReq("counter", "", nil))
	require.Error(t, err, "probe without a client cert should be rejected")
}

// TestServeServerTLS covers the server-only TLS mode (empty client CA): a
// client that presents no client cert is still accepted, and the channel is
// encrypted/authenticated against the server cert.
func TestServeServerTLS(t *testing.T) {
	caFile, certFile, keyFile := writeTestCerts(t)
	addr := freeTCPAddr(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := Serve(
			Listen(addr),
			ShutdownOn(ctx),
			TLS(certFile, keyFile, ""), // empty client CA => no client cert required
			Register("counter", counterProbe),
		)
		log.Println("Serve returned:", err)
	}()

	caPEM, err := os.ReadFile(caFile)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		ServerName: "localhost",
	})))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	client := pb.NewProberClient(conn)

	require.Eventually(t, func() bool {
		rpcCtx, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		_, err := client.Probe(rpcCtx, probeReq("counter", "", nil))
		return err == nil
	}, 5*time.Second, 50*time.Millisecond, "server-TLS probe never succeeded")
}

func TestTLSOptionError(t *testing.T) {
	err := Serve(
		Listen(freeTCPAddr(t)),
		TLS("/no/such/cert.pem", "/no/such/key.pem", ""),
		Register("counter", counterProbe),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading TLS cert/key")
}

func TestTLSOptionEmptyPaths(t *testing.T) {
	err := Serve(
		Listen(freeTCPAddr(t)),
		TLS("cert.pem", "", ""),
		Register("counter", counterProbe),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires both certFile and keyFile")
}

func TestIsLoopbackListen(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:9314":       true,
		"localhost:9314":       true,
		"[::1]:9314":           true,
		":9314":                false, // all interfaces
		"0.0.0.0:9314":         false,
		"10.0.0.5:9314":        false,
		"sidecar.example:9314": false,
		"not-an-addr":          false, // SplitHostPort error
	} {
		assert.Equalf(t, want, isLoopbackListen(addr), "isLoopbackListen(%q)", addr)
	}
}
