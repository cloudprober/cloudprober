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
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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
// being evicted and Closed. Default is 10 minutes. This is the backstop that
// keeps a sidecar from leaking sessions when targets disappear on the
// cloudprober side.
func IdleTTL(d time.Duration) Option {
	return func(s *server) { s.idleTTL = d }
}

// Register adds a probe type under the given name. A single sidecar process
// can serve any number of probe types.
func Register[C, S any](name string, pt ProbeType[C, S]) Option {
	return func(s *server) { s.probeTypes[name] = typedHandler[C, S]{pt: pt} }
}

type session struct {
	val      any
	h        handler
	lastUsed time.Time // guarded by server.mu
}

type server struct {
	pb.UnimplementedProberServer

	addr       string
	idleTTL    time.Duration
	probeTypes map[string]handler

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
	if s.addr == "" {
		return fmt.Errorf("no listen address; use sidecar.Listen()")
	}
	if len(s.probeTypes) == 0 {
		return fmt.Errorf("no probe types registered; use sidecar.Register()")
	}

	var lis net.Listener
	var err error
	if path, ok := strings.CutPrefix(s.addr, "unix://"); ok {
		os.Remove(path) // remove stale socket from a previous run
		lis, err = net.Listen("unix", path)
	} else {
		lis, err = net.Listen("tcp", s.addr)
	}
	if err != nil {
		return fmt.Errorf("listening on %s: %v", s.addr, err)
	}

	go s.sweepIdleSessions()

	grpcServer := grpc.NewServer()
	pb.RegisterProberServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, health.NewServer())

	var types []string
	for name := range s.probeTypes {
		types = append(types, name)
	}
	log.Printf("cloudprober sidecar: serving probe types %v on %s", types, s.addr)
	return grpcServer.Serve(lis)
}

// Probe implements the cloudprober.probes.grpcext.Prober service.
func (s *server) Probe(ctx context.Context, req *pb.ProbeRequest) (*pb.ProbeResponse, error) {
	h := s.probeTypes[req.GetProbeType()]
	if h == nil {
		return toProto(Internal(fmt.Errorf("unknown probe type: %q", req.GetProbeType())), nil), nil
	}

	t := Target{
		Name:   req.GetTarget().GetName(),
		Labels: req.GetTarget().GetLabels(),
		IP:     req.GetTarget().GetIp(),
		Port:   int(req.GetTarget().GetPort()),
	}

	var sessVal any
	var handle []byte
	if h.stateful() {
		var err error
		sessVal, handle, err = s.getSession(ctx, h, t, req)
		if err != nil {
			return toProto(Internal(fmt.Errorf("creating session: %v", err)), nil), nil
		}
	}

	return toProto(runProbeSafely(ctx, h, t, req.GetConfig(), sessVal), handle), nil
}

// getSession returns the session for the request's state handle, building a
// new one (and minting a new handle) if the handle is empty or unknown —
// e.g. on first contact, after sidecar restart, or after idle eviction.
//
// Note: sessions aren't single-flighted; cloudprober runs one probe loop per
// target, so concurrent first-probes for the same target don't happen in
// practice.
func (s *server) getSession(ctx context.Context, h handler, t Target, req *pb.ProbeRequest) (any, []byte, error) {
	handle := req.GetStateHandle()

	s.mu.Lock()
	if sess := s.sessions[string(handle)]; len(handle) > 0 && sess != nil {
		sess.lastUsed = time.Now()
		s.mu.Unlock()
		return sess.val, handle, nil
	}
	s.mu.Unlock()

	val, err := h.newSession(ctx, t, req.GetConfig())
	if err != nil {
		return nil, nil, err
	}

	handle = mintHandle()
	s.mu.Lock()
	s.sessions[string(handle)] = &session{val: val, h: h, lastUsed: time.Now()}
	s.mu.Unlock()
	log.Printf("cloudprober sidecar: new session for target %q (%s)", t.Name, req.GetProbeType())
	return val, handle, nil
}

func (s *server) sweepIdleSessions() {
	sweepInterval := s.idleTTL / 4
	if sweepInterval < time.Second {
		sweepInterval = time.Second
	}
	for range time.Tick(sweepInterval) {
		var evicted []*session
		s.mu.Lock()
		for key, sess := range s.sessions {
			if time.Since(sess.lastUsed) > s.idleTTL {
				evicted = append(evicted, sess)
				delete(s.sessions, key)
			}
		}
		s.mu.Unlock()
		for _, sess := range evicted {
			sess.h.closeSession(sess.val)
		}
		if len(evicted) > 0 {
			log.Printf("cloudprober sidecar: evicted %d idle session(s)", len(evicted))
		}
	}
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
	return []byte(hex.EncodeToString(b))
}
