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
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var zombieReaperOnce sync.Once

func runCommand(ctx context.Context, cmd *exec.Cmd) error {
	// If we are running as PID 1 (typical on k8s and docker), we start a
	// zombie reaper goroutine to clean up any child processes left behind.
	// This is like built-in init process.
	if os.Getpid() == 1 {
		// We use a sync.Once to make sure we only start one zombie reaper per
		// cloudprober process.
		zombieReaperOnce.Do(func() {
			go func() {
				for {
					time.Sleep(1 * time.Minute)
					// Run a loop to reap any zombies.
					var err error
					for err == nil {
						_, err = syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
					}
				}
			}()
		})
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
	return cmd.Wait()
}
