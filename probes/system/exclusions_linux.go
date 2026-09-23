// Copyright 2025-2026 The Cloudprober Authors.
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

//go:build linux
// +build linux

package system

// defaultExcludeMounts is the list of mount points or mount point prefixes
// that should be excluded from disk usage monitoring by default to avoid
// virtual mounts, container runtimes, and duplicated overlays.
var defaultExcludeMounts = []string{
	"/dev",
	"/sys",
	"/proc",
	"/run/netns",
	"/run/containerd",
	"/run/k3s",
	"/run/docker",
	"/snap",
	"/var/lib/docker/overlay2",
	"/var/lib/docker/overlay",
	"/var/lib/containerd/io.containerd",
	"/var/lib/kubelet/pods",
	"/var/lib/rancher/k3s/agent/containerd",
}
