// Copyright 2017-2024 The Cloudprober Authors.
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

/*
Package command implements functionality to run a command for probes.

This package is used by the external and browser probe types.
*/
package command

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/cloudprober/cloudprober/logger"
)

const (
	maxScannerTokenSize = 256 * 1024
)

func isPipeOrFileClosedError(err error) bool {
	if err == io.ErrClosedPipe {
		return true
	}
	if pe, ok := err.(*os.PathError); ok {
		if pe.Err == os.ErrClosed {
			return true
		}
	}
	return false
}

type Command struct {
	CmdLine                []string
	EnvVars                []string
	WorkDir                string
	ProcessStreamingOutput func([]byte)

	// We create a goroutine to wait for child processes to finish. This field
	// dictates how long will that goroutine wait for child processes to finish
	// before giving up. This is to avoid unbounded number of goroutines in case
	// child processes misbehave.
	ChildProcessWaitTime time.Duration
}

func (c *Command) setupStreaming(cmd *exec.Cmd, l *logger.Logger) error {
	stdout := make(chan []byte)
	stdoutR, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	go func() {
		defer close(stdout)
		defer stdoutR.Close()
		scanner := bufio.NewScanner(stdoutR)

		buf := make([]byte, 0, bufio.MaxScanTokenSize)
		scanner.Buffer(buf, maxScannerTokenSize)

		for scanner.Scan() {
			stdout <- scanner.Bytes()
		}
		if err := scanner.Err(); err != nil && !isPipeOrFileClosedError(err) {
			l.Errorf("Error reading from stdout: %v", err)
		}
	}()

	// Capture stderr
	stderrR, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stderrR.Close()
		scanner := bufio.NewScanner(stderrR)

		buf := make([]byte, 0, bufio.MaxScanTokenSize)
		scanner.Buffer(buf, maxScannerTokenSize)

		for scanner.Scan() {
			l.WarningAttrs("process stderr", slog.String("process_stderr", scanner.Text()), slog.String("process_path", c.CmdLine[0]))
		}
		if err := scanner.Err(); err != nil && !isPipeOrFileClosedError(err) {
			l.ErrorAttrs(fmt.Sprintf("Error reading from stderr: %v", err), slog.String("process_path", c.CmdLine[0]))
		}
	}()

	go func() {
		for line := range stdout {
			c.ProcessStreamingOutput(line)
		}
	}()

	return nil
}

func (c *Command) Execute(ctx context.Context, l *logger.Logger) (string, error) {
	if len(c.CmdLine) == 0 {
		return "", errors.New("no command specified)")
	}
	cmd := exec.CommandContext(ctx, c.CmdLine[0], c.CmdLine[1:]...)
	if c.EnvVars != nil {
		cmd.Env = append(append(cmd.Env, os.Environ()...), c.EnvVars...)
	}

	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	if c.ProcessStreamingOutput != nil {
		if err := c.setupStreaming(cmd, l); err != nil {
			return "", fmt.Errorf("error setting up stdout/stderr streaming: %v", err)
		}
	} else {
		cmd.Stdout, cmd.Stderr = &stdoutBuf, &stderrBuf
	}

	l.Debugf("Running command: %v", cmd)
	err := runCommand(ctx, cmd, c.ChildProcessWaitTime)

	if err != nil {
		stdout, stderr := stdoutBuf.String(), stderrBuf.String()
		stderrout := ""
		if stdout != "" || stderr != "" {
			stderrout = fmt.Sprintf(" Stdout: %s, Stderr: %s", stdout, stderr)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("external probe process died with the status (%s). Stderr: %s", exitErr.Error(), stderrout)
		} else {
			return "", fmt.Errorf("error executing the external program. Err: %v. Stderr: %s", err, stderrout)
		}
	}

	return stdoutBuf.String(), nil
}
