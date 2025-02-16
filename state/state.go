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

/*
Package runconfig stores cloudprober config that is specific to a single
invocation. e.g., servers injected by external cloudprober users.
*/
package state

import (
	"net/http"
	"sync"
	"time"

	rdsserver "github.com/cloudprober/cloudprober/internal/rds/server"
	"google.golang.org/grpc"
)

// state stores cloudprober config that is specific to a single invocation.
// e.g., servers injected by external cloudprober users.
type state struct {
	sync.RWMutex
	grpcSrv        *grpc.Server
	version        string
	buildTimestamp time.Time
	rdsServer      *rdsserver.Server
	httpServeMux   *http.ServeMux
	configFilePath string
}

var st state

// SetDefaultGRPCServer sets the default gRPC server.
func SetDefaultGRPCServer(s *grpc.Server) {
	st.Lock()
	defer st.Unlock()
	st.grpcSrv = s
}

// DefaultGRPCServer returns the configured gRPC server and nil if gRPC server
// was not set.
func DefaultGRPCServer() *grpc.Server {
	st.Lock()
	defer st.Unlock()
	return st.grpcSrv
}

// SetVersion sets the cloudprober version.
func SetVersion(version string) {
	st.Lock()
	defer st.Unlock()
	st.version = version
}

// Version returns the runconfig version set through the SetVersion() function
// call. It's useful only if called after SetVersion(), otherwise it will
// return an empty string.
func Version() string {
	st.RLock()
	defer st.RUnlock()
	return st.version
}

// SetBuildTimestamp sets the cloudprober build timestamp.
func SetBuildTimestamp(ts time.Time) {
	st.Lock()
	defer st.Unlock()
	st.buildTimestamp = ts
}

// BuildTimestamp returns the recorded build timestamp.
func BuildTimestamp() time.Time {
	st.RLock()
	defer st.RUnlock()
	return st.buildTimestamp
}

// SetLocalRDSServer stores local RDS server in the state. It can later
// be retrieved throuhg LocalRDSServer().
func SetLocalRDSServer(srv *rdsserver.Server) {
	st.Lock()
	defer st.Unlock()
	st.rdsServer = srv
}

// LocalRDSServer returns the local RDS server, set through the
// SetLocalRDSServer() call.
func LocalRDSServer() *rdsserver.Server {
	st.RLock()
	defer st.RUnlock()
	return st.rdsServer
}

// SetDefaultHTTPServeMux stores the default HTTP ServeMux in state. This
// allows other modules to add their own handlers to the common ServeMux.
func SetDefaultHTTPServeMux(mux *http.ServeMux) {
	st.Lock()
	defer st.Unlock()
	st.httpServeMux = mux
}

// DefaultHTTPServeMux returns the default HTTP ServeMux.
func DefaultHTTPServeMux() *http.ServeMux {
	st.RLock()
	defer st.RUnlock()
	return st.httpServeMux
}

func SetConfigFilePath(configFilePath string) {
	st.Lock()
	defer st.Unlock()
	st.configFilePath = configFilePath
}

func ConfigFilePath() string {
	st.RLock()
	defer st.RUnlock()
	return st.configFilePath
}
