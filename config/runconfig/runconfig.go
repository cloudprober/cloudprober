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
package runconfig

import (
	"net/http"
	"sync"
	"time"

	rdsserver "github.com/cloudprober/cloudprober/internal/rds/server"
	"google.golang.org/grpc"
)

// runConfig stores cloudprober config that is specific to a single invocation.
// e.g., servers injected by external cloudprober users.
type runConfig struct {
	sync.RWMutex
	grpcSrv        *grpc.Server
	version        string
	buildTimestamp time.Time
	rdsServer      *rdsserver.Server
	httpServeMux   *http.ServeMux
}

var rc runConfig

// SetDefaultGRPCServer sets the default gRPC server.
func SetDefaultGRPCServer(s *grpc.Server) {
	rc.Lock()
	defer rc.Unlock()
	rc.grpcSrv = s
}

// DefaultGRPCServer returns the configured gRPC server and nil if gRPC server
// was not set.
func DefaultGRPCServer() *grpc.Server {
	rc.Lock()
	defer rc.Unlock()
	return rc.grpcSrv
}

// SetVersion sets the cloudprober version.
func SetVersion(version string) {
	rc.Lock()
	defer rc.Unlock()
	rc.version = version
}

// Version returns the runconfig version set through the SetVersion() function
// call. It's useful only if called after SetVersion(), otherwise it will
// return an empty string.
func Version() string {
	rc.RLock()
	defer rc.RUnlock()
	return rc.version
}

// SetBuildTimestamp sets the cloudprober build timestamp.
func SetBuildTimestamp(ts time.Time) {
	rc.Lock()
	defer rc.Unlock()
	rc.buildTimestamp = ts
}

// BuildTimestamp returns the recorded build timestamp.
func BuildTimestamp() time.Time {
	rc.RLock()
	defer rc.RUnlock()
	return rc.buildTimestamp
}

// SetLocalRDSServer stores local RDS server in the runconfig. It can later
// be retrieved throuhg LocalRDSServer().
func SetLocalRDSServer(srv *rdsserver.Server) {
	rc.Lock()
	defer rc.Unlock()
	rc.rdsServer = srv
}

// LocalRDSServer returns the local RDS server, set through the
// SetLocalRDSServer() call.
func LocalRDSServer() *rdsserver.Server {
	rc.RLock()
	defer rc.RUnlock()
	return rc.rdsServer
}

// SetDefaultHTTPServeMux stores the default HTTP ServeMux in runconfig. This
// allows other modules to add their own handlers to the common ServeMux.
func SetDefaultHTTPServeMux(mux *http.ServeMux) {
	rc.Lock()
	defer rc.Unlock()
	rc.httpServeMux = mux
}

// DefaultHTTPServeMux returns the default HTTP ServeMux.
func DefaultHTTPServeMux() *http.ServeMux {
	rc.RLock()
	defer rc.RUnlock()
	return rc.httpServeMux
}
