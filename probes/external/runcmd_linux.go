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

package external

import (
	"bytes"
	"context"
	"os/exec"
	"syscall"
	"time"
)

func (p *Probe) runCommand(ctx context.Context, cmd string, args []string) ([]byte, error) {
	if p.runCommandFunc != nil {
		return p.runCommandFunc(ctx, cmd, args)
	}

	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	waitDone := make(chan struct{})
	defer close(waitDone)

	if err := c.Start(); err != nil {
		return stdout.Bytes(), err
	}

	go func() {
		select {
		case <-ctx.Done():
			// Kill the whole process group.
			syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		case <-waitDone:
			return
		}
	}()
	err := c.Wait()

	// Start a goroutine to wait on exited processes in the process group.
	go func() {
		var err error
		// Use timer to make sure we don't created unterminated goroutines.
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

	return stdout.Bytes(), err
}
