package state

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestDefaultGRPCServer(t *testing.T) {
	if srv := DefaultGRPCServer(); srv != nil {
		t.Fatalf("State has server unexpectedly set. Got %v Want nil", srv)
	}
	testSrv := grpc.NewServer()
	if testSrv == nil {
		t.Fatal("Unable to create a test gRPC server")
	}
	SetDefaultGRPCServer(testSrv)
	assert.Equal(t, testSrv, DefaultGRPCServer(), "gRPC server")
}

func TestHTTPServeMux(t *testing.T) {
	if mux := DefaultHTTPServeMux(); mux != nil {
		t.Fatalf("State has http serve mux unexpectedly set. Got %v Want nil", mux)
	}
	testMux := http.NewServeMux()
	if testMux == nil {
		t.Fatal("Unable to create a test http serve mux")
	}
	SetDefaultHTTPServeMux(testMux)
	assert.Equal(t, testMux, DefaultHTTPServeMux(), "http serve mux")
}

func TestConfigFilePath(t *testing.T) {
	SetConfigFilePath("/cfg/cloudprober.cfg")
	assert.Equal(t, "/cfg/cloudprober.cfg", ConfigFilePath(), "config file path")
}

func TestVersion(t *testing.T) {
	if v := Version(); v != "" {
		t.Fatalf("State has version unexpectedly set. Got %v Want \"\"", v)
	}
	SetVersion("v1.0.0")
	assert.Equal(t, "v1.0.0", Version(), "version")
}

func TestBuildTimestamp(t *testing.T) {
	if ts := BuildTimestamp(); !ts.IsZero() {
		t.Fatalf("State has build timestamp unexpectedly set. Got %v Want \"\"", ts)
	}
	ts := time.Now()
	SetBuildTimestamp(ts)
	assert.Equal(t, ts, BuildTimestamp(), "build timestamp")
}

func TestLocalRDSServer(t *testing.T) {
	if rds := LocalRDSServer(); rds != nil {
		t.Fatalf("State has LocalRDSServer unexpectedly set. Got %v Want nil", rds)
	}
}
