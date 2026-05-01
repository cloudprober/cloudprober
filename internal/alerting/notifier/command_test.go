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

package notifier

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// countZombieChildren returns the number of zombie children of the current
// process by walking /proc. Linux-only.
func countZombieChildren(t *testing.T) int {
	t.Helper()
	myPid := strconv.Itoa(os.Getpid())

	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatalf("read /proc: %v", err)
	}

	var zombies int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		statusBytes, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue // process may have disappeared
		}
		status := string(statusBytes)
		var ppid, state string
		for _, line := range strings.Split(status, "\n") {
			switch {
			case strings.HasPrefix(line, "PPid:"):
				ppid = strings.TrimSpace(strings.TrimPrefix(line, "PPid:"))
			case strings.HasPrefix(line, "State:"):
				state = strings.TrimSpace(strings.TrimPrefix(line, "State:"))
			}
		}
		if ppid == myPid && strings.HasPrefix(state, "Z") {
			zombies++
		}
	}
	return zombies
}

// TestCommandNotifierReapsChild verifies that the command notifier waits for
// the child process to exit, preventing zombie process accumulation.
// See https://github.com/cloudprober/cloudprober/issues/1298.
func TestCommandNotifierReapsChild(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("zombie detection via /proc is Linux-only")
	}

	cn, err := newCommandNotifier("/bin/true", nil)
	if err != nil {
		t.Fatalf("newCommandNotifier failed: %v", err)
	}

	const n = 10
	for i := 0; i < n; i++ {
		if err := cn.Notify(context.Background(), nil); err != nil {
			t.Fatalf("Notify returned error: %v", err)
		}
	}

	// Wait up to 2 seconds for all reaping goroutines to finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countZombieChildren(t) == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	if z := countZombieChildren(t); z > 0 {
		t.Fatalf("expected 0 zombie children, found %d", z)
	}
}
