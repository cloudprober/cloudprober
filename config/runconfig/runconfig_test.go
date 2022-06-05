package runconfig

import (
	"testing"

	"google.golang.org/grpc"
)

func TestRunConfig(t *testing.T) {
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
