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

// procID identifies a process by its pid and its start time. The start time is
// what keeps us from confusing a recycled pid with the zombie we first saw a
// minute ago: the kernel doesn't record when a process died, so all we have is
// when we first noticed it, and that's only sound if we can tell the two
// apart.
type procID struct {
	pid       int
	startTime uint64
}

type reaper struct {
	procRoot string
	pid      int
	interval time.Duration
	grace    time.Duration
	l        *logger.Logger

	// When we first saw a zombie child.
	seen map[procID]time.Time
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
		seen:     make(map[procID]time.Time),
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
	current := make(map[procID]bool, len(zombies))

	for _, proc := range zombies {
		current[proc] = true

		firstSeen, ok := r.seen[proc]
		if !ok {
			// First time we're seeing this one; give its owner, if it has one,
			// a chance to wait on it.
			r.seen[proc] = now
			continue
		}
		if now.Sub(firstSeen) < r.grace {
			continue
		}

		// syscall.ECHILD means somebody else got to it first, which is fine.
		if err := r.reap(proc.pid); err != nil && !errors.Is(err, syscall.ECHILD) {
			r.l.Warningf("Error reaping orphaned child process %d: %v", proc.pid, err)
			continue
		}
		delete(r.seen, proc)
		reaped = append(reaped, proc.pid)
	}

	// Forget the zombies that are gone now, whether we reaped them or their
	// owner did.
	for proc := range r.seen {
		if !current[proc] {
			delete(r.seen, proc)
		}
	}

	if len(reaped) > 0 {
		r.l.Infof("Reaped %d orphaned child process(es): %v", len(reaped), reaped)
	}
}

// zombieChildren returns our children that are in the zombie ("Z") state, by
// scanning /proc.
func (r *reaper) zombieChildren() ([]procID, error) {
	entries, err := os.ReadDir(r.procRoot)
	if err != nil {
		return nil, err
	}

	var zombies []procID
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a process directory.
		}
		b, err := os.ReadFile(filepath.Join(r.procRoot, entry.Name(), "stat"))
		if err != nil {
			continue // Process is gone already; nothing to reap.
		}
		state, ppid, startTime, err := parseStat(b)
		if err != nil {
			r.l.Warningf("Error parsing stat of the process %d: %v", pid, err)
			continue
		}
		if state == 'Z' && ppid == r.pid {
			zombies = append(zombies, procID{pid: pid, startTime: startTime})
		}
	}

	return zombies, nil
}

// parseStat returns the process state, parent pid, and start time -- the 3rd,
// 4th, and 22nd fields -- from the contents of /proc/<pid>/stat. Note that the
// 2nd field (comm) is the executable name in parentheses and can itself
// contain spaces and parentheses, so we start parsing after the last ")".
func parseStat(b []byte) (byte, int, uint64, error) {
	i := bytes.LastIndexByte(b, ')')
	if i < 0 {
		return 0, 0, 0, fmt.Errorf("unexpected format, no \")\" in %q", string(b))
	}

	// After comm, fields[0] is the 3rd field, so the 22nd is fields[19].
	fields := strings.Fields(string(b[i+1:]))
	if len(fields) < 20 {
		return 0, 0, 0, fmt.Errorf("unexpected format, only %d fields after comm in %q", len(fields), string(b))
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad ppid (%s): %v", fields[1], err)
	}

	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad start time (%s): %v", fields[19], err)
	}

	return fields[0][0], ppid, startTime, nil
}
