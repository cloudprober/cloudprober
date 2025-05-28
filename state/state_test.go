// Copyright 2018-2025 The Cloudprober Authors.
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

package state

import (
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
