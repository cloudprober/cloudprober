package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRunConfigDefaultGRPCServer(t *testing.T) {
	if srv := DefaultGRPCServer(); srv != nil {
		t.Fatalf("RunConfig has server unexpectedly set. Got %v Want nil", srv)
	}
	testSrv := grpc.NewServer()
	if testSrv == nil {
		t.Fatal("Unable to create a test gRPC server")
	}
	SetDefaultGRPCServer(testSrv)
	if srv := DefaultGRPCServer(); srv != testSrv {
		t.Fatalf("Error retrieving stored service. Got %v Want %v", srv, testSrv)
	}
}

func TestRunConfigConfigFilePath(t *testing.T) {
	SetConfigFilePath("/cfg/cloudprober.cfg")
	assert.Equal(t, "/cfg/cloudprober.cfg", ConfigFilePath(), "config file path")
}
