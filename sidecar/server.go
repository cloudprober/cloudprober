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
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	pb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

const defaultIdleTTL = 10 * time.Minute

// Option configures Serve.
type Option func(*server)

// Listen sets the address to serve on: "unix:///path/to/socket" for a unix
// domain socket (the co-located default), or a TCP address like ":9314".
func Listen(addr string) Option {
	return func(s *server) { s.addr = addr }
}

// IdleTTL overrides how long an unused per-target session is kept before
// being evicted and Closed. Default is 10 minutes; a zero or negative value
// disables idle eviction entirely. Sessions of probes that report their
// interval are additionally kept for at least 2x the interval, so
// long-interval probes don't lose their session between runs regardless of
// this setting. This is the backstop that keeps a sidecar from leaking
// sessions when targets disappear on the cloudprober side.
func IdleTTL(d time.Duration) Option {
	return func(s *server) { s.idleTTL = d }
}

// Register adds a probe type under the given name. A single sidecar process
// can serve any number of probe types. Registering a probe type without a
// Probe function, or two probe types under the same name, is an error that
// Serve reports.
func Register[C, S any](name string, pt ProbeType[C, S]) Option {
	return func(s *server) {
		if pt.Probe == nil {
			s.initErrs = append(s.initErrs, fmt.Errorf("probe type %q: Probe function is required", name))
			return
		}
		if _, dup := s.probeTypes[name]; dup {
			s.initErrs = append(s.initErrs, fmt.Errorf("probe type %q registered twice", name))
			return
		}
		s.probeTypes[name] = typedHandler[C, S]{pt: pt}
	}
}

type session struct {
	val      any
	h        handler
	typeName string

	// Fields below are guarded by server.mu.
	lastUsed time.Time
	inUse    int           // in-flight probes using this session
	evict    bool          // close and drop once inUse reaches 0
	minTTL   time.Duration // interval-derived TTL floor (2x probe interval)
}

type server struct {
	pb.UnimplementedProberServer

	addr       string
	idleTTL    time.Duration
	probeTypes map[string]handler
	initErrs   []error

	mu       sync.Mutex
	sessions map[string]*session // keyed by state handle
}

// Serve runs the sidecar server. It blocks until the listener fails.
func Serve(opts ...Option) error {
	s := &server{
		idleTTL:    defaultIdleTTL,
		probeTypes: make(map[string]handler),
		sessions:   make(map[string]*session),
	}
	for _, opt := range opts {
		opt(s)
	}
	if len(s.initErrs) > 0 {
		return errors.Join(s.initErrs...)
	}
	if s.addr == "" {
		return fmt.Errorf("no listen address; use sidecar.Listen()")
	}
	if len(s.probeTypes) == 0 {
		return fmt.Errorf("no probe types registered; use sidecar.Register()")
	}

	var lis net.Listener
	var err error
	if path, ok := strings.CutPrefix(s.addr, "unix://"); ok {
		if err := removeStaleSocket(path); err != nil {
			return err
		}
		lis, err = net.Listen("unix", path)
	} else {
		lis, err = net.Listen("tcp", s.addr)
	}
	if err != nil {
		return fmt.Errorf("listening on %s: %v", s.addr, err)
	}

	stopSweep := make(chan struct{})
	defer close(stopSweep)
	go s.sweepIdleSessions(stopSweep)

	grpcServer := grpc.NewServer()
	pb.RegisterProberServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, health.NewServer())

	log.Printf("cloudprober sidecar: serving probe types %v on %s", s.typeNames(), s.addr)
	return grpcServer.Serve(lis)
}

// removeStaleSocket clears a leftover socket file from a previous run, but
// refuses to remove a path that isn't a socket or that a live server is
// still accepting connections on.
func removeStaleSocket(path string) error {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove %s: it exists and is not a socket", path)
	}
	if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
		conn.Close()
		return fmt.Errorf("%s is in use by a running server", path)
	}
	return os.Remove(path)
}

func (s *server) typeNames() []string {
	var types []string
	for name := range s.probeTypes {
		types = append(types, name)
	}
	sort.Strings(types)
	return types
}

// Probe implements the cloudprober.probes.grpcext.Prober service.
func (s *server) Probe(ctx context.Context, req *pb.ProbeRequest) (*pb.ProbeResponse, error) {
	h := s.probeTypes[req.GetProbeType()]
	if h == nil {
		return toProto(Internal(fmt.Errorf("unknown probe type: %q (registered: %v)", req.GetProbeType(), s.typeNames())), nil), nil
	}

	t := Target{
		Name:   req.GetTarget().GetName(),
		Labels: req.GetTarget().GetLabels(),
		IP:     req.GetTarget().GetIp(),
		Port:   int(req.GetTarget().GetPort()),
	}

	var sess *session
	var handle []byte
	if h.stateful() {
		var errRes *Result
		sess, handle, errRes = s.acquireSession(ctx, h, t, req)
		if errRes != nil {
			return toProto(errRes, nil), nil
		}
		defer s.releaseSession(string(handle), sess)
	}

	var sessVal any
	if sess != nil {
		sessVal = sess.val
	}
	res := runProbeSafely(ctx, h, t, req.GetConfig(), sessVal)
	if res.invalidateSession && sess != nil {
		s.mu.Lock()
		sess.evict = true
		s.mu.Unlock()
	}
	return toProto(res, handle), nil
}

// ValidateConfig implements the config check cloudprober runs at probe init.
func (s *server) ValidateConfig(ctx context.Context, req *pb.ValidateConfigRequest) (*pb.ValidateConfigResponse, error) {
	h := s.probeTypes[req.GetProbeType()]
	if h == nil {
		return nil, status.Errorf(codes.NotFound, "unknown probe type: %q (registered: %v)", req.GetProbeType(), s.typeNames())
	}
	if err := h.validateConfig(req.GetConfig()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "probe type %q: %v", req.GetProbeType(), err)
	}
	return &pb.ValidateConfigResponse{}, nil
}

// acquireSession returns the session for the request's state handle, pinned
// against eviction until releaseSession. A new session (and handle) is built
// if the handle is empty, unknown (first contact, sidecar restart, idle
// eviction), or belongs to a different probe type. On failure it returns an
// error Result: config-decode errors are internal (misconfiguration), while
// New errors are target-level failures — for connection-per-target probe
// types, New failing IS "can't reach the target".
//
// Note: sessions aren't single-flighted; cloudprober runs one probe loop per
// target, so concurrent first-probes for the same target don't happen in
// practice.
func (s *server) acquireSession(ctx context.Context, h handler, t Target, req *pb.ProbeRequest) (*session, []byte, *Result) {
	handle := req.GetStateHandle()
	minTTL := 2 * req.GetInterval().AsDuration()

	s.mu.Lock()
	if sess := s.sessions[string(handle)]; len(handle) > 0 && sess != nil && sess.typeName == req.GetProbeType() {
		sess.lastUsed = time.Now()
		sess.inUse++
		sess.minTTL = minTTL
		s.mu.Unlock()
		return sess, handle, nil
	}
	s.mu.Unlock()

	val, err := h.newSession(ctx, t, req.GetConfig())
	if err != nil {
		var ce configError
		if errors.As(err, &ce) {
			return nil, nil, Internal(err)
		}
		return nil, nil, Fail(fmt.Errorf("creating session: %v", err))
	}

	handle = mintHandle()
	sess := &session{val: val, h: h, typeName: req.GetProbeType(), lastUsed: time.Now(), inUse: 1, minTTL: minTTL}
	s.mu.Lock()
	s.sessions[string(handle)] = sess
	s.mu.Unlock()
	log.Printf("cloudprober sidecar: new session for target %q (%s)", t.Name, req.GetProbeType())
	return sess, handle, nil
}

func (s *server) releaseSession(key string, sess *session) {
	s.mu.Lock()
	sess.inUse--
	sess.lastUsed = time.Now()
	evictNow := sess.evict && sess.inUse == 0 && s.sessions[key] == sess
	if evictNow {
		delete(s.sessions, key)
	}
	s.mu.Unlock()
	if evictNow {
		closeSafely(sess)
	}
}

func (s *server) sweepIdleSessions(stop <-chan struct{}) {
	sweepInterval := s.idleTTL / 4
	if sweepInterval < time.Second {
		sweepInterval = time.Second
	}
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		var evicted []*session
		s.mu.Lock()
		for key, sess := range s.sessions {
			ttl := s.idleTTL
			if sess.minTTL > ttl {
				ttl = sess.minTTL
			}
			// idleTTL <= 0 means idle eviction is disabled; explicit
			// invalidation (sess.evict) still applies.
			expired := s.idleTTL > 0 && time.Since(sess.lastUsed) > ttl
			if sess.inUse == 0 && (expired || sess.evict) {
				evicted = append(evicted, sess)
				delete(s.sessions, key)
			}
		}
		s.mu.Unlock()
		for _, sess := range evicted {
			closeSafely(sess)
		}
		if len(evicted) > 0 {
			log.Printf("cloudprober sidecar: evicted %d idle session(s)", len(evicted))
		}
	}
}

// closeSafely runs the author's Close hook, containing panics so a buggy
// Close can't take down the sidecar (which may serve other probe types).
func closeSafely(sess *session) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("cloudprober sidecar: panic in session Close: %v", r)
		}
	}()
	sess.h.closeSession(sess.val)
}

// runProbeSafely runs the probe handler, converting panics and nil results
// into internal errors so a buggy probe type can't take down the whole
// sidecar (which may be serving other probe types).
func runProbeSafely(ctx context.Context, h handler, t Target, config []byte, session any) (res *Result) {
	defer func() {
		if r := recover(); r != nil {
			res = Internal(fmt.Errorf("probe panicked: %v", r))
		}
	}()
	if res = h.runProbe(ctx, t, config, session); res == nil {
		res = Internal(fmt.Errorf("probe returned nil result"))
	}
	return res
}

func toProto(r *Result, stateHandle []byte) *pb.ProbeResponse {
	resp := &pb.ProbeResponse{
		Success:       r.success,
		InternalError: r.internal,
		Latency:       durationpb.New(r.latency),
		Payload:       r.payload,
		StateHandle:   stateHandle,
	}
	if r.err != nil {
		resp.Error = r.err.Error()
	}
	return resp
}

func mintHandle() []byte {
	b := make([]byte, 16)
	rand.Read(b)
	return b
}
