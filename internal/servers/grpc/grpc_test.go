// Copyright 2018 The Cloudprober Authors.
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

// Unit tests for grpc server.
package grpc

import (
	"context"
	"errors"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	pb "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	spb "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/state"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var once sync.Once

// globalGRPCServer sets up runconfig and returns a gRPC server.
func globalGRPCServer() (*grpc.Server, error) {
	once.Do(func() {
		state.SetDefaultGRPCServer(grpc.NewServer())
	})
	srv := state.DefaultGRPCServer()
	if srv == nil {
		return nil, errors.New("runconfig gRPC server not setup properly")
	}
	return srv, nil
}

func TestGRPCSuccess(t *testing.T) {
	// Skip this test if HTTP proxy environment variables are set. Some
	// corporate environments intercept outbound local TCP traffic causing
	// tests that dial loopback addresses to fail with HTTP 407.
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" ||
		os.Getenv("HTTPS_PROXY") != "" || os.Getenv("https_proxy") != "" {
		t.Skip("Skipping gRPC tests due to proxy environment variables")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// global config setup is necessary for gRPC probe server.
	if _, err := globalGRPCServer(); err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}
	cfg := &configpb.ServerConf{
		Port: proto.Int32(0),
	}
	l := &logger.Logger{}

	srv, err := New(ctx, cfg, l)
	if err != nil {
		t.Fatalf("Unable to create grpc server: %v", err)
	}
	go srv.Start(ctx, nil)
	if !srv.dedicatedSrv {
		t.Error("Probe server not using dedicated gRPC server.")
	}

	listenAddr := srv.ln.Addr().String()
	conn, err := grpc.Dial(listenAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Unable to connect to grpc server at %v: %v", listenAddr, err)
	}

	client := spb.NewProberClient(conn)
	timedCtx, timedCancel := context.WithTimeout(ctx, time.Second)
	defer timedCancel()
	sReq := &pb.StatusRequest{}
	sResp, err := client.ServerStatus(timedCtx, sReq)
	if err != nil {
		t.Errorf("ServerStatus call error: %v", err)
	}
	t.Logf("Uptime: %v", sResp.GetUptimeUs())
	if sResp.GetUptimeUs() == 0 {
		t.Error("Uptime not being incremented.")
	}

	timedCtx, timedCancel = context.WithTimeout(ctx, time.Second)
	defer timedCancel()
	msg := []byte("test message")
	echoReq := &pb.EchoMessage{Blob: msg}
	echoResp, err := client.Echo(timedCtx, echoReq)
	if err != nil {
		t.Errorf("Echo call error: %v", err)
	}
	t.Logf("EchoResponse: <%v>", string(echoResp.Blob))
	if !reflect.DeepEqual(echoResp.Blob, echoReq.Blob) {
		t.Errorf("Echo response mismatch: got %v want %v", echoResp.Blob, echoReq.Blob)
	}

	wantReadSize := 4
	readReq := &pb.BlobReadRequest{Size: proto.Int32(int32(wantReadSize))}
	readResp, err := client.BlobRead(timedCtx, readReq)
	if err != nil {
		t.Errorf("Read call error: %v", err)
	}
	t.Logf("ReadResponse: <%v>", readResp)
	readSize := len(readResp.GetBlob())
	if readSize != wantReadSize {
		t.Errorf("Read response mismatch: got %v want %v", readSize, wantReadSize)
	}

	msg = []byte("test_write")
	writeReq := &pb.BlobWriteRequest{Blob: msg}
	writeResp, err := client.BlobWrite(timedCtx, writeReq)
	if err != nil {
		t.Errorf("Write call error: %v", err)
	}
	t.Logf("WriteResponse: <%v>", writeResp)
	if writeResp.GetSize() != int32(len(msg)) {
		t.Errorf("Write response mismatch: got %v want %v", writeResp.GetSize(), len(msg))
	}
}

func TestInjection(t *testing.T) {
	// Skip this test if HTTP proxy environment variables are set. Some
	// corporate environments intercept outbound local TCP traffic causing
	// tests that dial loopback addresses to fail with HTTP 407.
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" ||
		os.Getenv("HTTPS_PROXY") != "" || os.Getenv("https_proxy") != "" {
		t.Skip("Skipping gRPC injection test due to proxy environment variables")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcSrv, err := globalGRPCServer()
	if err != nil {
		t.Fatalf("Error getting global gRPC server: %v", err)
	}
	cfg := &configpb.ServerConf{
		Port:               proto.Int32(0),
		UseDedicatedServer: proto.Bool(false),
	}
	l := &logger.Logger{}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Unable to open socket for listening: %v", err)
	}
	defer ln.Close()

	if _, err = New(ctx, cfg, l); err != nil {
		t.Fatalf("Error creating gRPC probe server: %v", err)
	}
	go grpcSrv.Serve(ln)
	time.Sleep(time.Second)

	listenAddr := ln.Addr().String()
	conn, err := grpc.Dial(listenAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Unable to connect to grpc server at %v: %v", listenAddr, err)
	}

	client := spb.NewProberClient(conn)
	timedCtx, timedCancel := context.WithTimeout(ctx, time.Second)
	defer timedCancel()
	sReq := &pb.StatusRequest{}
	sResp, err := client.ServerStatus(timedCtx, sReq)
	if err != nil {
		t.Errorf("ServerStatus call error: %v", err)
	}
	t.Logf("Uptime: %v", sResp.GetUptimeUs())
	if sResp.GetUptimeUs() == 0 {
		t.Error("Uptime not being incremented.")
	}
}

func TestInjectionOverride(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcSrv, err := globalGRPCServer()
	if err != nil {
		t.Fatalf("Error getting global gRPC server: %v", err)
	}
	cfg := &configpb.ServerConf{
		Port: proto.Int32(0),
	}
	l := &logger.Logger{}

	srv, err := New(ctx, cfg, l)
	if err != nil {
		t.Fatalf("Error creating gRPC probe server: %v", err)
	}
	if srv.grpcSrv == grpcSrv {
		t.Error("Probe server not using dedicated gRPC server.")
	}
	if !srv.dedicatedSrv {
		t.Error("Got dedicatedSrv=false, want true.")
	}
}

func TestSizeError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// global config setup is necessary for gRPC probe server.
	if _, err := globalGRPCServer(); err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}
	cfg := &configpb.ServerConf{
		Port: proto.Int32(0),
	}
	l := &logger.Logger{}

	srv, err := New(ctx, cfg, l)
	if err != nil {
		t.Fatalf("Unable to create grpc server: %v", err)
	}
	go srv.Start(ctx, nil)
	if !srv.dedicatedSrv {
		t.Error("Probe server not using dedicated gRPC server.")
	}

	listenAddr := srv.ln.Addr().String()
	conn, err := grpc.Dial(listenAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Unable to connect to grpc server at %v: %v", listenAddr, err)
	}

	client := spb.NewProberClient(conn)
	timedCtx, timedCancel := context.WithTimeout(ctx, time.Second)
	defer timedCancel()

	readReq := &pb.BlobReadRequest{Size: proto.Int32(int32(maxMsgSize + 1))}
	readResp, err := client.BlobRead(timedCtx, readReq)
	if err == nil {
		t.Errorf("Read call unexpectedly succeeded: %v", readResp)
	}

	writeReq := &pb.BlobWriteRequest{Blob: make([]byte, maxMsgSize+1)}
	writeResp, err := client.BlobWrite(timedCtx, writeReq)
	if err == nil {
		t.Errorf("Write call unexpectedly succeeded: %v", writeResp)
	}

}
