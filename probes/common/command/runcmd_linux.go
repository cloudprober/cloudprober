// Copyright 2022-2024 The Cloudprober Authors.
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

// This file defines Linux specific implementaion of runCommand. We don't use
// the standard exec.CommandOutput method as it doesn't provide a way to clean
// up the further processes started by the command. We start the given command
// in a new process group, and kill the whole process group on time out.
// Background: https://github.com/cloudprober/cloudprober/issues/165.

package command

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

var defaultChildProcessWaitTime = 10 * time.Second

// How long to wait between wait4 calls while the process group still has
// running processes in it.
const childProcessPollInterval = 100 * time.Millisecond

func runCommand(ctx context.Context, cmd *exec.Cmd, childProcessWaitTime time.Duration) error {
	if childProcessWaitTime == 0 {
		childProcessWaitTime = defaultChildProcessWaitTime
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return err
	}

	// This goroutine is similar to the one started by exec.Start if command is
	// created with exec.CommandContext(..). The difference is that we kill the
	// whole process group instead of just one process.
	waitDone := make(chan struct{})
	defer close(waitDone)
	go func() {
		select {
		case <-ctx.Done():
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		case <-waitDone:
			return
		}
	}()
	err := cmd.Wait()

	// Start a goroutine to wait on the processes in the process group, to
	// avoid zombies. We give up after childProcessWaitTime to make sure we
	// don't create an unbounded number of goroutines in case a command hangs
	// even on SIGKILL.
	// Note that this covers only the processes still in our process group;
	// anything that started a group of its own (browsers, for example) and
	// then got reparented to us is handled by internal/reaper.
	go func() {
		// The deadline bounds the loop as a whole, reaps included: a process
		// group that keeps producing dead children shouldn't keep this
		// goroutine -- and we start one per run -- alive indefinitely.
		// Whatever is left when we give up is picked up by internal/reaper,
		// if we're the process collecting orphans.
		deadline := time.Now().Add(childProcessWaitTime)

		for time.Now().Before(deadline) {
			// An error here (ECHILD) means there is nothing left in the
			// process group to wait for.
			pid, err := syscall.Wait4(-cmd.Process.Pid, nil, syscall.WNOHANG, nil)
			if err != nil {
				return
			}
			// We reaped one; there may be more that are ready right now.
			if pid > 0 {
				continue
			}
			// Nothing has exited yet; check back in a bit instead of spinning
			// on wait4 for the whole childProcessWaitTime.
			time.Sleep(childProcessPollInterval)
		}
	}()

	return err
}
