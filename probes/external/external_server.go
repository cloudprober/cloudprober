// Copyright 2017-2023 The Cloudprober Authors.
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
Package external implements an external probe type for cloudprober.

External probe type executes an external process for actual probing. These probes
can have two modes: "once" and "server". In "once" mode, the external process is
started for each probe run cycle, while in "server" mode, external process is
started only if it's not running already and Cloudprober communicates with it
over stdin/stdout for each probe cycle.
*/
package external

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	configpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/cloudprober/cloudprober/probes/external/serverutils"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"google.golang.org/protobuf/proto"
)

var (
	// TimeBetweenRequests is the time interval between probe requests for
	// multiple targets. In server mode, probe requests for multiple targets are
	// sent to the same external probe process. Sleeping between requests provides
	// some time buffer for the probe process to dequeue the incoming requests and
	// avoids filling up the communication pipe.
	//
	// Note that this value impacts the effective timeout for a target as timeout
	// is applied for all the targets in aggregate. For example, 100th target in
	// the targets list will have the effective timeout of (timeout - 1ms).
	// TODO(manugarg): Make sure that the last target in the list has an impact of
	// less than 1% on its timeout.
	TimeBetweenRequests = 10 * time.Microsecond
)

// monitorCommand waits for the process to terminate and sets cmdRunning to
// false when that happens.
func (p *Probe) monitorCommand(startCtx context.Context, cmd commandIntf) error {
	err := cmd.Wait()

	// Dont'log error message if killed explicitly.
	if startCtx.Err() != nil {
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), string(exitErr.Stderr))
	}
	return err
}

func (p *Probe) startCmdIfNotRunning(startCtx context.Context) error {
	// Start external probe command if it's not running already. Note that here we
	// are trusting the cmdRunning to be set correctly. It can be false for 4
	// reasons:
	// Correct reasons:
	// 1) This is the first call and process has actually never been started.
	// 2) Process died for some reason and monitor set cmdRunning to false.
	// Incorrect reasons:
	// 3) cmd.Start() started the process but still returned an error.
	// 4) cmd.Wait() returned incorrectly, while the process was still running.
	//
	// 3 or 4 should never really happen, but managing processes can be tricky.
	// Documenting here to help with debugging if we run into an issue.
	p.cmdRunningMu.Lock()
	defer p.cmdRunningMu.Unlock()
	if p.cmdRunning {
		return nil
	}
	p.l.Infof("Starting external command: %s %s", p.cmdName, strings.Join(p.cmdArgs, " "))
	cmd := exec.CommandContext(startCtx, p.cmdName, p.cmdArgs...)
	var err error
	if p.cmdStdin, err = cmd.StdinPipe(); err != nil {
		return err
	}
	if p.cmdStdout, err = cmd.StdoutPipe(); err != nil {
		return err
	}
	if p.cmdStderr, err = cmd.StderrPipe(); err != nil {
		return err
	}
	if len(p.envVars) > 0 {
		cmd.Env = append(cmd.Env, p.envVars...)
	}

	go func() {
		scanner := bufio.NewScanner(p.cmdStderr)
		for {
			if scanner.Scan() {
				p.l.WarningAttrs("process stderr", slog.String("process_stderr", scanner.Text()), slog.String("process_path", cmd.Path))
				continue
			}
			if scanner.Err() == bufio.ErrTooLong {
				// Drop the associated log but otherwise continue reading.
				p.l.Errorf("stderr reader: encountered ErrTooLong from %s", cmd.Path)
				scanner = bufio.NewScanner(p.cmdStderr)
				continue
			}
			p.l.Warningf("Stderr reader: encountered terminal error from %s", cmd.Path, scanner.Err())
			break
		}
	}()

	if err = cmd.Start(); err != nil {
		p.l.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
		return fmt.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
	}

	ctx, cancelReadProbeReplies := context.WithCancel(startCtx)
	// This goroutine waits for the process to terminate and sets cmdRunning to
	// false when that happens.
	go func() {
		if err := p.monitorCommand(startCtx, cmd); err != nil {
			p.l.Error(err.Error())
		}
		cancelReadProbeReplies()
		p.cmdRunningMu.Lock()
		p.cmdRunning = false
		p.cmdRunningMu.Unlock()
	}()
	go p.readProbeReplies(ctx)
	p.cmdRunning = true
	return nil
}

func (p *Probe) readProbeReplies(ctx context.Context) error {
	bufReader := bufio.NewReader(p.cmdStdout)
	// Start a background goroutine to read probe replies from the probe server
	// process's stdout and put them on the probe's replyChan. Note that replyChan
	// is a one element channel. Idea is that we won't need buffering other than
	// the one provided by Unix pipes.
	for {
		if ctx.Err() != nil {
			return nil
		}
		rep := new(configpb.ProbeReply)
		if err := serverutils.ReadMessage(ctx, rep, bufReader); err != nil {
			// Return if external probe process pipe has closed. We get:
			//  io.EOF: when other process has closed the pipe.
			//  os.ErrClosed: when we have closed the pipe (through cmd.Wait()).
			// *os.PathError: deferred close of the pipe.
			_, isPathError := err.(*os.PathError)
			if err == os.ErrClosed || err == io.EOF || isPathError {
				p.l.Errorf("External probe process pipe is closed. Err: %s", err.Error())
				return err
			}
			p.l.Errorf("Error reading probe reply: %s", err.Error())
			continue
		}
		p.replyChan <- rep
	}

}

func (p *Probe) sendRequest(requestID int32, ep endpoint.Endpoint) error {
	req := &configpb.ProbeRequest{
		RequestId: proto.Int32(requestID),
		TimeLimit: proto.Int32(int32(p.opts.Timeout / time.Millisecond)),
		Options:   []*configpb.ProbeRequest_Option{},
	}
	for _, opt := range p.c.GetOptions() {
		value := opt.GetValue()
		if len(p.labelKeys) != 0 { // If we're looking for substitions.
			res, found := strtemplate.SubstituteLabels(value, p.labels(ep))
			if !found {
				p.l.Warningf("Missing substitution in option %q", value)
			} else {
				value = res
			}
		}
		req.Options = append(req.Options, &configpb.ProbeRequest_Option{
			Name:  opt.Name,
			Value: proto.String(value),
		})
	}

	p.l.Debugf("Sending a probe request %v to the external probe server for target %v", requestID, ep.Name)
	return serverutils.WriteMessage(req, p.cmdStdin)
}

func (p *Probe) runServerProbe(ctx, startCtx context.Context) {
	type requestInfo struct {
		target    endpoint.Endpoint
		timestamp time.Time
	}

	outstandingReqs := make(map[int32]requestInfo)
	var outstandingReqsMu sync.RWMutex

	if err := p.startCmdIfNotRunning(startCtx); err != nil {
		p.l.Error(err.Error())
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Read probe replies until we have no outstanding requests or context has
		// run out.
		expectedRepliesReceived := 0
		for {
			select {
			case <-ctx.Done():
				p.l.Error(ctx.Err().Error())
				return
			case rep := <-p.replyChan:
				outstandingReqsMu.Lock()
				reqInfo, ok := outstandingReqs[rep.GetRequestId()]
				if ok {
					expectedRepliesReceived++
					delete(outstandingReqs, rep.GetRequestId())
				}
				outstandingReqsMu.Unlock()
				if !ok {
					// Not our reply, could be from the last timed out probe.
					p.l.Warningf("Got a reply that doesn't match any outstading request: Request id from reply: %v. Ignoring.", rep.GetRequestId())
					continue
				}
				success := true
				if rep.GetErrorMessage() != "" {
					p.l.Errorf("Probe for target %v failed with error message: %s", reqInfo.target, rep.GetErrorMessage())
					success = false
				}
				ps := &probeStatus{
					success: success,
					latency: time.Since(reqInfo.timestamp),
					payload: rep.GetPayload(),
				}
				p.processProbeResult(ps, reqInfo.target, p.results[reqInfo.target.Key()])
			}

			// We send a total if len(p.targets) requests. We can exit if we've
			// seen replies for all of them.
			if expectedRepliesReceived == len(p.targets) {
				return
			}
		}
	}()

	// Send probe requests
	for _, target := range p.targets {
		p.requestID++
		p.results[target.Key()].total++
		outstandingReqsMu.Lock()
		outstandingReqs[p.requestID] = requestInfo{
			target:    target,
			timestamp: time.Now(),
		}
		outstandingReqsMu.Unlock()
		p.sendRequest(p.requestID, target)
		time.Sleep(TimeBetweenRequests)
	}

	// Wait for receiver goroutine to exit.
	wg.Wait()

	// Handle requests that we have not yet received replies for: "requests" will
	// contain only outstanding requests by this point.
	outstandingReqsMu.Lock()
	defer outstandingReqsMu.Unlock()
	for _, req := range outstandingReqs {
		p.processProbeResult(&probeStatus{success: false}, req.target, p.results[req.target.Key()])
	}
}
