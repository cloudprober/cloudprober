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

package external

import (
	"bufio"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics/testutils"
	configpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/cloudprober/cloudprober/probes/external/serverutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// startProbeServer starts a test probe server to work with the TestProbeServer
// test below.
func startProbeServer(t *testing.T, ctx context.Context, testPayload string, r io.Reader, w io.WriteCloser) {
	rd := bufio.NewReader(r)
	for {
		if ctx.Err() != nil {
			return
		}

		req := &configpb.ProbeRequest{}
		if err := serverutils.ReadMessage(context.Background(), req, rd); err != nil {
			if ctx.Err() != nil {
				return
			}
			t.Errorf("Error reading probe request. Err: %v", err)
			return
		}
		var action, target string
		opts := req.GetOptions()
		for _, opt := range opts {
			if opt.GetName() == "action" {
				action = opt.GetValue()
				continue
			}
			if opt.GetName() == "target" {
				target = opt.GetValue()
				continue
			}
		}
		id := req.GetRequestId()

		actionToResponse := map[string]*configpb.ProbeReply{
			"nopayload": {RequestId: proto.Int32(id)},
			"payload": {
				RequestId: proto.Int32(id),
				Payload:   proto.String(testPayload),
			},
			"payload_with_error": {
				RequestId:    proto.Int32(id),
				Payload:      proto.String(testPayload),
				ErrorMessage: proto.String("error"),
			},
		}
		t.Logf("Request id: %d, action: %s, target: %s", id, action, target)
		if action == "pipe_server_close" {
			w.Close()
			return
		}
		if res, ok := actionToResponse[action]; ok {
			serverutils.WriteMessage(res, w)
		}
	}
}

// runAndVerifyServerProbe executes a server probe and verifies the replies
// received.
func runAndVerifyServerProbe(t *testing.T, p *Probe, action string, tgts []string, total, success map[string]int64, numEventMetrics int) {
	setProbeOptions(p, "action", action)

	runAndVerifyProbe(t, p, tgts, total, success)

	// Verify that we got all the expected EventMetrics
	ems, err := testutils.MetricsFromChannel(p.dataChan, numEventMetrics, 1*time.Second)
	if err != nil {
		t.Error(err)
	}
	mmap := testutils.MetricsMapByTarget(ems)

	for _, tgt := range tgts {
		assert.Equal(t, total[tgt], mmap.LastValueInt64(tgt, "total"))
		assert.Equal(t, success[tgt], mmap.LastValueInt64(tgt, "success"))
	}
}

func testProbeServerSetup(t *testing.T, ctx context.Context, readErrorCh chan error) (*Probe, string) {
	// We create two pairs of pipes to establish communication between this prober
	// and the test probe server (defined above).
	// Test probe server input pipe. We writes on w1 and external command reads
	// from r1.
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Errorf("Error creating OS pipe. Err: %v", err)
	}
	// Test probe server output pipe. External command writes on w2 and we read
	// from r2.
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Errorf("Error creating OS pipe. Err: %v", err)
	}

	testPayload := "p90 45\n"
	// Start probe server in a goroutine
	go startProbeServer(t, ctx, testPayload, r1, w2)

	p := createTestProbe("./testCommand", nil, configpb.ProbeConf_SERVER)
	p.cmdRunning = true // don't try to start the probe server
	p.cmdStdin = w1
	p.cmdStdout = r2

	// Start the goroutine that reads probe replies.
	go func() {
		err := p.readProbeReplies(ctx)
		if readErrorCh != nil {
			readErrorCh <- err
			close(readErrorCh)
		}
	}()

	return p, testPayload
}

func TestProbeServerMode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p, _ := testProbeServerSetup(t, ctx, nil)
	defer cancel()

	total, success := make(map[string]int64), make(map[string]int64)

	// No payload
	tgts := []string{"target1", "target2"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	t.Run("nopayload", func(t *testing.T) {
		runAndVerifyServerProbe(t, p, "nopayload", tgts, total, success, 2)
	})

	// Payload
	tgts = []string{"target3"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	t.Run("payload", func(t *testing.T) {
		// 2 metrics per target
		runAndVerifyServerProbe(t, p, "payload", tgts, total, success, 1*2)
	})

	// Payload with error
	tgts = []string{"target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	t.Run("payload_with_error", func(t *testing.T) {
		// 2 targets, 2 EMs per target
		runAndVerifyServerProbe(t, p, "payload_with_error", tgts, total, success, 2*2)
	})

	// Timeout
	tgts = []string{"target1", "target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}

	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	t.Run("timeout", func(t *testing.T) {
		// 3 targets, 1 EM per target
		runAndVerifyServerProbe(t, p, "timeout", tgts, total, success, 3*1)
	})
}

func TestProbeServerRemotePipeClose(t *testing.T) {
	readErrorCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	p, _ := testProbeServerSetup(t, ctx, readErrorCh)
	defer cancel()

	total, success := make(map[string]int64), make(map[string]int64)
	// Remote pipe close
	tgts := []string{"target"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	runAndVerifyServerProbe(t, p, "pipe_server_close", tgts, total, success, 1)
	readError := <-readErrorCh
	if readError == nil {
		t.Error("Didn't get error in reading pipe")
	}
	if readError != io.EOF {
		t.Errorf("Didn't get correct error in reading pipe. Got: %v, wanted: %v", readError, io.EOF)
	}
}

func TestProbeServerLocalPipeClose(t *testing.T) {
	readErrorCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	p, _ := testProbeServerSetup(t, ctx, readErrorCh)
	defer cancel()

	total, success := make(map[string]int64), make(map[string]int64)
	// Local pipe close
	tgts := []string{"target"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	p.cmdStdout.(*os.File).Close()
	runAndVerifyServerProbe(t, p, "pipe_local_close", tgts, total, success, 1)
	readError := <-readErrorCh
	if readError == nil {
		t.Error("Didn't get error in reading pipe")
	}
	if _, ok := readError.(*os.PathError); !ok {
		t.Errorf("Didn't get correct error in reading pipe. Got: %T, wanted: *os.PathError", readError)
	}
}
