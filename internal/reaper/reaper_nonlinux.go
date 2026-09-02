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

//go:build !linux

// Package reaper reaps orphaned child processes that get reparented to
// cloudprober. It's implemented only on Linux; see reaper_linux.go.
package reaper

import (
	"context"

	"github.com/cloudprober/cloudprober/logger"
)

// Start is a no-op on non-Linux platforms.
func Start(_ context.Context, _ *logger.Logger) {
	// Nothing to do here. Reaping orphans is built on procfs and wait4, and
	// the deployments that make cloudprober init for a process tree -- our
	// container images -- are Linux-only anyway.
}
