// Copyright 2020 The Cloudprober Authors.
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

package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tlsconfigpb "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	configpb "github.com/cloudprober/cloudprober/probes/grpc/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	pb "github.com/cloudprober/cloudprober/servers/grpc/proto"
	spb "github.com/cloudprober/cloudprober/servers/grpc/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/cloudprober/cloudprober/targets/resolver"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

var once sync.Once
var srvAddr string

type Server struct {
	delay time.Duration
	msg   []byte

	// Required for all gRPC server implementations.
	spb.UnimplementedProberServer
}

// Echo reflects back the incoming message.
// TODO: return error if EchoMessage is greater than maxMsgSize.
func (s *Server) Echo(ctx context.Context, req *pb.EchoMessage) (*pb.EchoMessage, error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return req, nil
}

// BlobRead returns a blob of data.
func (s *Server) BlobRead(ctx context.Context, req *pb.BlobReadRequest) (*pb.BlobReadResponse, error) {
	return &pb.BlobReadResponse{
		Blob: s.msg[0:req.GetSize()],
	}, nil
}

// ServerStatus returns the current server status.
func (s *Server) ServerStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{
		UptimeUs: proto.Int64(42),
	}, nil
}

// BlobWrite returns the size of blob in the WriteRequest. It does not operate
// on the blob.
func (s *Server) BlobWrite(ctx context.Context, req *pb.BlobWriteRequest) (*pb.BlobWriteResponse, error) {
	return &pb.BlobWriteResponse{
		Size: proto.Int32(int32(len(req.Blob))),
	}, nil
}

// globalGRPCServer sets up runconfig and returns a gRPC server.
func globalGRPCServer(delay time.Duration) (string, error) {
	var err error
	once.Do(func() {
		var ln net.Listener
		ln, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return
		}
		grpcSrv := grpc.NewServer()
		srv := &Server{delay: delay, msg: make([]byte, 1024)}
		spb.RegisterProberServer(grpcSrv, srv)
		go grpcSrv.Serve(ln)
		tcpAddr := ln.Addr().(*net.TCPAddr)
		srvAddr = net.JoinHostPort(tcpAddr.IP.String(), strconv.Itoa(tcpAddr.Port))
		time.Sleep(time.Second * 2)
	})
	return srvAddr, err
}

// TestGRPCSuccess tests probe output on success.
// 2 connections, 1 probe/sec/conn, stats exported every 5 sec
// 	=> 5-10 results/interval. Test looks for minimum of 7 results.
func TestGRPCSuccess(t *testing.T) {
	interval, timeout := 100*time.Millisecond, 100*time.Millisecond
	addr, err := globalGRPCServer(timeout / 2)
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}

	iters := 5
	statsExportInterval := time.Duration(iters) * interval

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Interval:            interval,
		Timeout:             timeout,
		ProbeConf:           &configpb.ProbeConf{NumConns: proto.Int32(2)},
		Logger:              &logger.Logger{},
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc-success", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 5)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	expectedLabels := map[string]string{"ptype": "grpc", "dst": addr, "probe": "grpc-success"}

	ems, err := testutils.MetricsFromChannel(dataChan, 2, 1500*time.Millisecond)
	if err != nil || len(ems) != 2 {
		t.Errorf("Err: %v", err)
	}

	for i, em := range ems {
		expectedMinCount := int64((i + 1) * (iters + 1))
		assert.GreaterOrEqual(t, em.Metric("total").(*metrics.Int).Int64(), expectedMinCount, "message#: %d, total, em: %s", i, em.String())
		assert.GreaterOrEqual(t, em.Metric("success").(*metrics.Int).Int64(), expectedMinCount, "message#: %d, success, em: %s", i, em.String())
		gotLabels := make(map[string]string)
		for _, k := range em.LabelsKeys() {
			gotLabels[k] = em.Label(k)
		}
		assert.Equal(t, expectedLabels, gotLabels)
	}

	cancel()
	wg.Wait()
}

// TestConnectFailures attempts to connect to localhost:9 (discard port) and
// checks that stats are exported once every connect timeout.
// 2 connections, 0.5 connect attempt/sec/conn, stats exported every 6 sec
//  => 3 - 6 connect errors/sec. Test looks for minimum of 4 attempts.
func TestConnectFailures(t *testing.T) {
	interval, timeout := 100*time.Millisecond, 100*time.Millisecond
	addr := "localhost:9"

	iters := 6
	statsExportInterval := time.Duration(6) * interval

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Interval:            interval,
		Timeout:             timeout,
		ProbeConf:           &configpb.ProbeConf{NumConns: proto.Int32(2)},
		Logger:              &logger.Logger{},
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc-connectfail", probeOpts)
	p.dialOpts = append(p.dialOpts, grpc.WithBlock())
	dataChan := make(chan *metrics.EventMetrics, 5)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	ems, err := testutils.MetricsFromChannel(dataChan, 2, 1500*time.Millisecond)
	if err != nil || len(ems) != 2 {
		t.Errorf("Err: %v", err)
	}

	for i, em := range ems {
		expectedMinCount := int64((i + 1) * (iters + 1))
		assert.GreaterOrEqual(t, em.Metric("total").(*metrics.Int).Int64(), expectedMinCount, "message#: %d, total, em: %s", i, em.String())
		assert.GreaterOrEqual(t, em.Metric("connecterrors").(*metrics.Int).Int64(), expectedMinCount, "message#: %d, connecterrors, em: %s", i, em.String())
		// 0 success
		assert.Equal(t, int64(0), em.Metric("success").(*metrics.Int).Int64(), "message#: %d, success, em: %s", i, em.String())
	}

	cancel()
	wg.Wait()
}

func TestProbeTimeouts(t *testing.T) {
	interval, timeout := 100*time.Millisecond, 10*time.Millisecond

	addr, err := globalGRPCServer(timeout * 2)
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}

	iters := 5
	statsExportInterval := time.Duration(iters) * interval

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Interval:            interval,
		Timeout:             timeout,
		ProbeConf:           &configpb.ProbeConf{NumConns: proto.Int32(1)},
		Logger:              &logger.Logger{},
		LatencyUnit:         time.Millisecond,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}

	p := &Probe{}
	p.Init("grpc-reqtimeout", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 5)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	ems, err := testutils.MetricsFromChannel(dataChan, 2, 3*statsExportInterval)
	if err != nil || len(ems) != 2 {
		t.Errorf("Err: %v", err)
	}

	for i, em := range ems {
		expectedMinCount := int64((i + 1) * (iters/2 + 1))
		assert.GreaterOrEqual(t, em.Metric("total").(*metrics.Int).Int64(), expectedMinCount, "message#: %d, total, em: %s", i, em.String())
		// 0 success
		assert.Equal(t, int64(0), em.Metric("success").(*metrics.Int).Int64(), "message#: %d, success, em: %s", i, em.String())
	}

	cancel()
	wg.Wait()
}

type testTargets struct {
	r *resolver.Resolver

	start        time.Time
	startTargets []endpoint.Endpoint

	switchDur   time.Duration
	nextTargets []endpoint.Endpoint
}

func newTargets(startTargets, nextTargets []endpoint.Endpoint, switchDur time.Duration) targets.Targets {
	return &testTargets{r: resolver.New(), startTargets: startTargets, nextTargets: nextTargets, start: time.Now(), switchDur: switchDur}
}

func (t *testTargets) ListEndpoints() []endpoint.Endpoint {
	if time.Since(t.start) > t.switchDur {
		return t.nextTargets
	}
	return t.startTargets
}

func (t *testTargets) Resolve(name string, ipVer int) (net.IP, error) {
	return t.r.Resolve(name, ipVer)
}

func TestTargets(t *testing.T) {
	interval, timeout := 100*time.Millisecond, 100*time.Millisecond

	addr, err := globalGRPCServer(timeout / 2)
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}

	// Target discovery changes from good to bad targets after 2 statsExports.
	// And probe continues for 10 more stats exports.
	statsExportInterval := 1 * interval
	probeRunTime := 12 * interval

	TargetsUpdateInterval = 2 * interval
	badTargets := targets.StaticTargets("localhost:1,localhost:2").ListEndpoints()
	goodTargets := targets.StaticTargets(addr).ListEndpoints()
	// This targets switches from bad targets to good targets after 1 interval.
	tgts := newTargets(goodTargets, badTargets, TargetsUpdateInterval-interval)

	probeOpts := &options.Options{
		Targets:             tgts,
		Timeout:             timeout,
		Interval:            interval,
		ProbeConf:           &configpb.ProbeConf{NumConns: proto.Int32(2)},
		LatencyUnit:         time.Millisecond,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}

	p := &Probe{}
	p.Init("grpc", probeOpts)
	p.dialOpts = append(p.dialOpts, grpc.WithBlock())
	dataChan := make(chan *metrics.EventMetrics, 10)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	ems, err := testutils.MetricsFromChannel(dataChan, 12, probeRunTime)
	if err != nil {
		t.Fatalf("Error retrieving metrics: %v", err)
	}
	mm := testutils.MetricsMap(ems)

	sumIntMetrics := func(ems []*metrics.EventMetrics, metricName string) int64 {
		sum := metrics.NewInt(0)
		for _, em := range ems {
			sum.Add(em.Metric(metricName))
		}
		return sum.Int64()
	}

	connErrTargets := make(map[string]int64)
	connErrIterCount := 0
	for target, vals := range mm["connecterrors"] {
		s := sumIntMetrics(vals, "connecterrors")
		if s > 0 {
			connErrTargets[target] = s
		}
		if len(vals) > connErrIterCount {
			connErrIterCount = len(vals)
		}
	}

	successTargets := make(map[string]int64)
	successIterCount := 0
	for target, vals := range mm["success"] {
		s := sumIntMetrics(vals, "success")
		if s > 0 {
			successTargets[target] = s
			if connErrTargets[target] > 0 {
				t.Errorf("Target %s has both success and failures.", target)
			}
			if len(vals) > successIterCount {
				successIterCount = len(vals)
			}
		}
	}

	assert.GreaterOrEqual(t, len(successTargets), 0, "zero targets with success, want at least one.")
	assert.GreaterOrEqual(t, len(connErrTargets), 0, "zero targets with conn errors, want at least one.")
	if successIterCount >= connErrIterCount {
		t.Errorf("Got successIters(%d) >= connErrIters(%d), want '<'.", successIterCount, connErrIterCount)
	}

	cancel()
	wg.Wait()
}

func TestHealthCheckProbe(t *testing.T) {
	response := map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
		"service-A": grpc_health_v1.HealthCheckResponse_SERVING,
		"service-B": grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}
	tests := []struct {
		service      string
		ignoreStatus bool
		wantErr      bool
		errText      string
	}{
		{
			service: "service-A",
		},
		{
			service: "service-B",
			wantErr: true,
			errText: "NOT_SERVING",
		},
		{
			service:      "service-B",
			ignoreStatus: true,
			wantErr:      false,
		},
		{
			service: "service-err",
			wantErr: true,
			errText: "service-err",
		},
		{
			service:      "service-err",
			ignoreStatus: true,
			wantErr:      true,
			errText:      "service-err",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-ignoreStatus:%v", test.service, test.ignoreStatus), func(t *testing.T) {
			p := &Probe{
				c: &configpb.ProbeConf{
					HealthCheckIgnoreStatus: proto.Bool(test.ignoreStatus),
					HealthCheckService:      proto.String(test.service),
				},
			}

			p.healthCheckFunc = func() (*grpc_health_v1.HealthCheckResponse, error) {
				if test.service == "service-err" {
					return nil, errors.New(test.service)
				}
				return &grpc_health_v1.HealthCheckResponse{
					Status: response[test.service],
				}, nil
			}

			err := p.healthCheckProbe(context.Background(), nil, "")
			if err != nil && !test.wantErr {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if err == nil && test.wantErr {
				t.Error("Expected error but got none")
				return
			}
			if test.errText != "" && !strings.Contains(err.Error(), test.errText) {
				t.Errorf("Error (%s) doesn't contain expected error text (%s)", err.Error(), test.errText)
			}
		})
	}
}

func TestTransportCreds(t *testing.T) {
	tests := []struct {
		name     string
		c        *configpb.ProbeConf
		wantInfo string
		wantErr  bool
	}{
		{
			name:     "no transport config",
			c:        &configpb.ProbeConf{},
			wantInfo: "",
		},
		{
			name: "insecure_transport",
			c: &configpb.ProbeConf{
				InsecureTransport: proto.Bool(true),
			},
			wantInfo: "insecure",
		},
		{
			name: "tls_transport",
			c: &configpb.ProbeConf{
				TlsConfig: &tlsconfigpb.TLSConfig{},
			},
			wantInfo: "tls",
		},
		{
			name: "alts_transport",
			c: &configpb.ProbeConf{
				AltsConfig: &configpb.ProbeConf_ALTSConfig{},
			},
			wantInfo: "alts",
		},
		{
			name: "error_tls_and_alts_transport",
			c: &configpb.ProbeConf{
				TlsConfig:  &tlsconfigpb.TLSConfig{},
				AltsConfig: &configpb.ProbeConf_ALTSConfig{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				c: tt.c,
			}
			got, err := p.transportCredentials()
			if (err != nil) != tt.wantErr {
				t.Errorf("Probe.getTransportCreds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantInfo == "" {
				return
			}
			assert.Equal(t, tt.wantInfo, got.Info().SecurityProtocol)
		})
	}
}
