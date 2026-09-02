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

/*
Package reaper reaps orphaned child processes that get reparented to
cloudprober.

Our container images run cloudprober itself as the entrypoint, which makes it
PID 1 inside the container's PID namespace, and hence init: the kernel
reparents orphaned processes to it, and it inherits init's duty of waiting on
them. Processes that cloudprober starts itself are waited on by whoever started
them (through os/exec), but their descendants are not. The browser probe is the
worst offender: it runs "npx playwright test", playwright starts the browser in
its own process group (it spawns it detached), and the browser forks further
helper processes of its own. Whenever one of those outlives its parent, it is
reparented to us, and with nobody waiting on it, it stays around as a zombie
forever -- a handful of <defunct> browser processes per probe run, eventually
exhausting the PID space.

We can't just call wait4(-1): that would race with os/exec and steal the exit
statuses of processes cloudprober started itself, making those probe runs fail.
Instead we look for zombie children in /proc and reap only the ones that have
been sitting unclaimed for gracePeriod. Everything cloudprober starts is waited
on right away (all our callers block in cmd.Wait), so a zombie that is still
around after that long has no owner coming for it.

This only runs if orphans are actually being reparented to us, i.e. we're init
in our PID namespace or a child subreaper. Anything else means somebody else is
already doing this job -- /pause with shareProcessNamespace on Kubernetes, tini
with docker's --init -- and setting one of those up remains a perfectly good
way to sidestep this mechanism entirely.
*/
package reaper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/cloudprober/cloudprober/logger"
)

const (
	// How often we look for zombie children.
	scanInterval = 30 * time.Second

	// How long a zombie child has to stay unclaimed before we assume it's an
	// orphan and reap it. This is what keeps us from racing with os/exec over
	// the processes cloudprober starts itself; it's deliberately much longer
	// than the milliseconds it takes cmd.Wait to reap them.
	gracePeriod = 60 * time.Second

	// PR_GET_CHILD_SUBREAPER, from linux/prctl.h.
	prGetChildSubreaper = 37
)

type reaper struct {
	procRoot string
	pid      int
	interval time.Duration
	grace    time.Duration
	l        *logger.Logger

	// When we first saw a zombie child, keyed by pid.
	seen map[int]time.Time
	// Reaps the given pid; overridden in tests.
	reap func(pid int) error
}

// isSubreaper returns whether we've been marked a child subreaper, which gives
// us the same orphan-collecting duty as init. Cloudprober never sets this on
// itself, but the attribute survives execve, so a supervisor could have set it
// before starting us.
func isSubreaper() bool {
	var v int32
	if _, _, errno := syscall.Syscall(syscall.SYS_PRCTL, prGetChildSubreaper, uintptr(unsafe.Pointer(&v)), 0); errno != 0 {
		return false
	}
	return v != 0
}

// Start starts a background goroutine that reaps orphaned child processes that
// have been reparented to us. It's a no-op unless we're the one collecting
// orphans in the first place -- if we're neither init in our PID namespace nor
// a child subreaper, orphans go to somebody else, who is responsible for
// reaping them.
func Start(ctx context.Context, l *logger.Logger) {
	if os.Getpid() != 1 && !isSubreaper() {
		l.Debugf("Not init or a subreaper; orphaned processes are not ours to reap.")
		return
	}

	l.Infof("Collecting orphaned processes (pid: %d); starting the child process reaper.", os.Getpid())

	r := &reaper{
		procRoot: "/proc",
		pid:      os.Getpid(),
		interval: scanInterval,
		grace:    gracePeriod,
		l:        l,
		seen:     make(map[int]time.Time),
		reap:     reapPid,
	}
	go r.run(ctx)
}

func reapPid(pid int) error {
	_, err := syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
	return err
}

func (r *reaper) run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reapOrphans(time.Now())
		}
	}
}

func (r *reaper) reapOrphans(now time.Time) {
	zombies, err := r.zombieChildren()
	if err != nil {
		r.l.Warningf("Error looking for zombie child processes: %v", err)
		return
	}

	var reaped []int
	current := make(map[int]bool, len(zombies))

	for _, pid := range zombies {
		current[pid] = true

		firstSeen, ok := r.seen[pid]
		if !ok {
			// First time we're seeing this one; give its owner, if it has one,
			// a chance to wait on it.
			r.seen[pid] = now
			continue
		}
		if now.Sub(firstSeen) < r.grace {
			continue
		}

		// syscall.ECHILD means somebody else got to it first, which is fine.
		if err := r.reap(pid); err != nil && !errors.Is(err, syscall.ECHILD) {
			r.l.Warningf("Error reaping orphaned child process %d: %v", pid, err)
			continue
		}
		reaped = append(reaped, pid)
	}

	// Forget the zombies that are gone now, whether we reaped them or their
	// owner did.
	for pid := range r.seen {
		if !current[pid] {
			delete(r.seen, pid)
		}
	}
	for _, pid := range reaped {
		delete(r.seen, pid)
	}

	if len(reaped) > 0 {
		r.l.Infof("Reaped %d orphaned child process(es): %v", len(reaped), reaped)
	}
}

// zombieChildren returns the pids of our children that are in the zombie ("Z")
// state, by scanning /proc.
func (r *reaper) zombieChildren() ([]int, error) {
	entries, err := os.ReadDir(r.procRoot)
	if err != nil {
		return nil, err
	}

	var zombies []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a process directory.
		}
		b, err := os.ReadFile(filepath.Join(r.procRoot, entry.Name(), "stat"))
		if err != nil {
			continue // Process is gone already; nothing to reap.
		}
		state, ppid, err := parseStat(b)
		if err != nil {
			r.l.Warningf("Error parsing stat of the process %d: %v", pid, err)
			continue
		}
		if state == 'Z' && ppid == r.pid {
			zombies = append(zombies, pid)
		}
	}

	return zombies, nil
}

// parseStat returns the process state and parent pid -- the 3rd and 4th fields
// -- from the contents of /proc/<pid>/stat. Note that the 2nd field (comm) is
// the executable name in parentheses and can itself contain spaces and
// parentheses, so we start parsing after the last ")".
func parseStat(b []byte) (byte, int, error) {
	i := bytes.LastIndexByte(b, ')')
	if i < 0 {
		return 0, 0, fmt.Errorf("unexpected format, no \")\" in %q", string(b))
	}

	fields := strings.Fields(string(b[i+1:]))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected format, no state and ppid in %q", string(b))
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("bad ppid (%s): %v", fields[1], err)
	}

	return fields[0][0], ppid, nil
}
