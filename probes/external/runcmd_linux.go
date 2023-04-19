// Copyright 2022 The Cloudprober Authors.
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

package external

import (
	"bytes"
	"context"
	"os/exec"
	"syscall"
	"time"
)

func (p *Probe) runCommand(ctx context.Context, cmd string, args []string) ([]byte, []byte, error) {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	if err := c.Start(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}

	// This goroutine is similar to the one started by exec.Start if command is
	// created with exec.CommandContext(..). The difference is that we kill the
	// whole process group instead of just one process.
	waitDone := make(chan struct{})
	defer close(waitDone)
	go func() {
		select {
		case <-ctx.Done():
			syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		case <-waitDone:
			return
		}
	}()
	err := c.Wait()

	// Start a goroutine to wait on the processes in the process group, to
	// avoid zombies. We use a timer to make sure we don't create an
	// unbounded number of goroutines in case a command hangs even on SIGKILL.
	go func() {
		var err error

		timeout := time.NewTimer(time.Second)
		defer timeout.Stop()
		for err == nil {
			select {
			case <-timeout.C:
				return
			default:
			}
			_, err = syscall.Wait4(-c.Process.Pid, nil, syscall.WNOHANG, nil)
		}
	}()

	return stdout.Bytes(), stderr.Bytes(), err
}
