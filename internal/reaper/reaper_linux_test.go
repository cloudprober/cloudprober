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

//go:build linux

package reaper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PR_SET_CHILD_SUBREAPER, from linux/prctl.h.
const prSetChildSubreaper = 36

func TestParseStat(t *testing.T) {
	tests := []struct {
		desc      string
		stat      string
		wantState byte
		wantPPID  int
		wantErr   bool
	}{
		{
			desc:      "simple",
			stat:      "1234 (headless_shell) Z 1 1234 1234 0 -1 4194560 0 0",
			wantState: 'Z',
			wantPPID:  1,
		},
		{
			desc:      "comm with spaces and parens",
			stat:      "42 (node (js) worker) S 17 42 42 0 -1 4194304 0 0",
			wantState: 'S',
			wantPPID:  17,
		},
		{
			desc:    "no comm",
			stat:    "1234 Z 1 1234",
			wantErr: true,
		},
		{
			desc:    "truncated after comm",
			stat:    "1234 (sh) Z",
			wantErr: true,
		},
		{
			desc:    "bad ppid",
			stat:    "1234 (sh) Z init 1234",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			state, ppid, err := parseStat([]byte(test.stat))
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, string(test.wantState), string(state), "state")
			assert.Equal(t, test.wantPPID, ppid, "ppid")
		})
	}
}

// testReaper sets up a reaper over a fake /proc, with the given processes,
// each one described as "<state> <ppid>".
func testReaper(t *testing.T, procs map[int]string) *reaper {
	t.Helper()

	procRoot := t.TempDir()
	for pid, proc := range procs {
		dir := filepath.Join(procRoot, fmt.Sprintf("%d", pid))
		require.NoError(t, os.Mkdir(dir, 0755))
		stat := fmt.Sprintf("%d (headless_shell) %s 0 0 0 -1 4194560", pid, proc)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0644))
	}
	// Non-process entries in /proc should be ignored.
	require.NoError(t, os.Mkdir(filepath.Join(procRoot, "sys"), 0755))

	return &reaper{
		procRoot: procRoot,
		pid:      100,
		grace:    time.Minute,
		l:        &logger.Logger{},
		seen:     make(map[int]time.Time),
	}
}

func TestZombieChildren(t *testing.T) {
	r := testReaper(t, map[int]string{
		100: "S 1",   // us
		101: "Z 100", // zombie child
		102: "S 100", // running child
		103: "Z 100", // zombie child
		104: "Z 1",   // zombie, but not ours
	})

	zombies, err := r.zombieChildren()
	assert.NoError(t, err)
	sort.Ints(zombies)
	assert.Equal(t, []int{101, 103}, zombies)
}

func TestReapOrphansWaitsForGracePeriod(t *testing.T) {
	r := testReaper(t, map[int]string{
		101: "Z 100",
		102: "S 100",
	})

	var reaped []int
	r.reap = func(pid int) error {
		reaped = append(reaped, pid)
		return nil
	}

	now := time.Now()

	// First sighting: we only note the zombie, its owner may still wait on it.
	r.reapOrphans(now)
	assert.Empty(t, reaped, "reaped on first sighting")
	assert.Contains(t, r.seen, 101)

	// Still within the grace period.
	r.reapOrphans(now.Add(r.grace - time.Second))
	assert.Empty(t, reaped, "reaped within the grace period")

	// Unclaimed for longer than the grace period: it's an orphan.
	r.reapOrphans(now.Add(r.grace + time.Second))
	assert.Equal(t, []int{101}, reaped)
	assert.NotContains(t, r.seen, 101, "reaped pid not removed from seen")
}

func TestReapOrphansForgetsGoneZombies(t *testing.T) {
	r := testReaper(t, map[int]string{101: "Z 100"})
	r.reap = func(pid int) error { return nil }

	r.reapOrphans(time.Now())
	assert.Contains(t, r.seen, 101, "zombie not recorded")

	// Zombie's owner reaped it before our grace period ran out.
	require.NoError(t, os.RemoveAll(filepath.Join(r.procRoot, "101")))

	r.reapOrphans(time.Now())
	assert.NotContains(t, r.seen, 101, "vanished zombie not forgotten")
}

func TestReapOrphansIgnoresECHILD(t *testing.T) {
	r := testReaper(t, map[int]string{101: "Z 100"})
	r.grace = 0
	r.reap = func(pid int) error { return syscall.ECHILD }

	now := time.Now()
	r.reapOrphans(now)
	r.reapOrphans(now.Add(time.Second))

	// ECHILD means somebody else got to it first; we're done with this pid.
	assert.NotContains(t, r.seen, 101)
}

// TestReapRealZombie verifies the whole thing against a real zombie child and
// the real /proc.
func TestReapRealZombie(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	require.NoError(t, cmd.Start())
	pid := cmd.Process.Pid

	r := &reaper{
		procRoot: "/proc",
		pid:      os.Getpid(),
		grace:    0,
		l:        &logger.Logger{},
		seen:     make(map[int]time.Time),
		reap:     reapPid,
	}

	// Wait for the child to exit and become a zombie -- we never wait on it.
	assert.Eventually(t, func() bool {
		zombies, err := r.zombieChildren()
		if err != nil {
			return false
		}
		for _, z := range zombies {
			if z == pid {
				return true
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond, "child (%d) didn't show up as a zombie", pid)

	// First scan notes it, second one reaps it (grace period is zero here).
	r.reapOrphans(time.Now())
	r.reapOrphans(time.Now())

	zombies, err := r.zombieChildren()
	assert.NoError(t, err)
	assert.NotContains(t, zombies, pid, "zombie child was not reaped")
}

// TestOrphanedDetachedGrandchild reproduces what the browser probe runs into:
// a grandchild that setsid'ed itself into a process group of its own (that's
// how playwright starts browsers) is orphaned onto us, and we are the one
// collecting orphans. Nothing but this reaper can wait on such a process.
//
// We can't be PID 1 in a test, so we make ourselves a child subreaper instead,
// which gives us the same duty. That's a process-wide, inherited setting, so
// this runs in a subprocess.
func TestOrphanedDetachedGrandchild(t *testing.T) {
	if os.Getenv("GO_CP_TEST_SUBREAPER") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=TestOrphanedDetachedGrandchild", "-test.v")
		cmd.Env = append(os.Environ(), "GO_CP_TEST_SUBREAPER=1")
		out, err := cmd.CombinedOutput()
		assert.NoError(t, err, "subprocess output:\n%s", out)
		return
	}

	assert.False(t, isSubreaper(), "subreaper before we set it")

	// PR_SET_CHILD_SUBREAPER (36): orphaned descendants are reparented to us
	// instead of to init, exactly like they are to cloudprober as PID 1.
	_, _, errno := syscall.Syscall(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0)
	require.Zero(t, errno, "prctl(PR_SET_CHILD_SUBREAPER)")

	// This is what makes Start run the reaper for a process that isn't PID 1.
	assert.True(t, isSubreaper(), "subreaper after we set it")

	// The outer shell exits right away, orphaning the setsid'ed grandchild.
	cmd := exec.Command("/bin/sh", "-c", "setsid /bin/sh -c 'sleep 0.2' & exit 0")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, cmd.Run())

	r := &reaper{
		procRoot: "/proc",
		pid:      os.Getpid(),
		grace:    0,
		l:        &logger.Logger{},
		seen:     make(map[int]time.Time),
		reap:     reapPid,
	}

	// The grandchild is not in any process group we know of, so waiting on the
	// command's process group (what probes/common/command does) can't see it;
	// it lands on us as a zombie and stays one until we reap it.
	assert.Eventually(t, func() bool {
		zombies, _ := r.zombieChildren()
		return len(zombies) > 0
	}, 5*time.Second, 10*time.Millisecond, "orphaned grandchild never showed up as our zombie child")

	// First scan notes it, second one reaps it (grace period is zero here).
	r.reapOrphans(time.Now())
	r.reapOrphans(time.Now())

	zombies, err := r.zombieChildren()
	assert.NoError(t, err)
	assert.Empty(t, zombies, "zombies left after reaping")
}
