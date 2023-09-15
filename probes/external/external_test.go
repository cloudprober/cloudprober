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

package external

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	payloadconfigpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	configpb "github.com/cloudprober/cloudprober/probes/external/proto"
	serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/cloudprober/cloudprober/probes/external/serverutils"
	"github.com/cloudprober/cloudprober/probes/options"
	probeconfigpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func isDone(doneChan chan struct{}) bool {
	// If we are done, return immediately.
	select {
	case <-doneChan:
		return true
	default:
	}
	return false
}

// startProbeServer starts a test probe server to work with the TestProbeServer
// test below.
func startProbeServer(t *testing.T, testPayload string, r io.Reader, w io.WriteCloser, doneChan chan struct{}) {
	rd := bufio.NewReader(r)
	for {
		if isDone(doneChan) {
			return
		}

		req, err := serverutils.ReadProbeRequest(rd)
		if err != nil {
			// Normal failure because we are finished.
			if isDone(doneChan) {
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

		actionToResponse := map[string]*serverpb.ProbeReply{
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

func setProbeOptions(p *Probe, name, value string) {
	for _, opt := range p.c.Options {
		if opt.GetName() == name {
			opt.Value = proto.String(value)
			break
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

func runAndVerifyProbe(t *testing.T, p *Probe, tgts []string, total, success map[string]int64) {
	p.opts.Targets = targets.StaticTargets(strings.Join(tgts, ","))
	p.updateTargets()

	p.runProbe(context.Background())

	for _, target := range p.targets {
		tgtKey := target.Key()

		assert.Equal(t, total[target.Name], p.results[tgtKey].total, "total")
		assert.Equal(t, success[target.Name], p.results[tgtKey].success, "success")
	}
}

func createTestProbe(cmd string, envVar map[string]string) *Probe {
	probeConf := &configpb.ProbeConf{
		Options: []*configpb.ProbeConf_Option{
			{
				Name:  proto.String("action"),
				Value: proto.String(""),
			},
		},
		Command: &cmd,
		EnvVar:  envVar,
	}

	p := &Probe{
		dataChan: make(chan *metrics.EventMetrics, 20),
	}

	p.Init("testProbe", &options.Options{
		ProbeConf:         probeConf,
		Timeout:           1 * time.Second,
		LogMetrics:        func(em *metrics.EventMetrics) {},
		LatencyMetricName: "latency",
	})

	return p
}

func testProbeServerSetup(t *testing.T, readErrorCh chan error) (*Probe, string, chan struct{}) {
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
	doneChan := make(chan struct{})
	go startProbeServer(t, testPayload, r1, w2, doneChan)

	p := createTestProbe("./testCommand", nil)
	p.cmdRunning = true // don't try to start the probe server
	p.cmdStdin = w1
	p.cmdStdout = r2
	p.mode = "server"

	// Start the goroutine that reads probe replies.
	go func() {
		err := p.readProbeReplies(doneChan)
		if readErrorCh != nil {
			readErrorCh <- err
			close(readErrorCh)
		}
	}()

	return p, testPayload, doneChan
}

func TestProbeServerMode(t *testing.T) {
	p, _, doneChan := testProbeServerSetup(t, nil)
	defer close(doneChan)

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
	p, _, doneChan := testProbeServerSetup(t, readErrorCh)
	defer close(doneChan)

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
	p, _, doneChan := testProbeServerSetup(t, readErrorCh)
	defer close(doneChan)

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

func testProbeOnceMode(t *testing.T, cmd string, tgts []string, envVars map[string]string, wantCmd, wantEnv string, wantArgs []string) {
	t.Helper()

	p := createTestProbe(cmd, envVars)
	p.mode = "once"

	// Set runCommand to a function that runs successfully and returns a pyload.
	p.runCommandFunc = func(ctx context.Context, cmd string, cmdArgs, envVars []string) ([]byte, []byte, error) {
		var resp []string
		resp = append(resp, fmt.Sprintf("cmd \"%s\"", cmd))
		resp = append(resp, fmt.Sprintf("args \"%s\"", strings.Join(cmdArgs, ",")))
		resp = append(resp, fmt.Sprintf("env \"%s\"", strings.Join(envVars, " ")))
		return []byte(strings.Join(resp, "\n")), nil, nil
	}

	total, success := make(map[string]int64), make(map[string]int64)

	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}

	runAndVerifyProbe(t, p, tgts, total, success)

	// Try with failing command now
	p.runCommandFunc = func(ctx context.Context, cmd string, cmdArgs []string, envVars []string) ([]byte, []byte, error) {
		return nil, nil, fmt.Errorf("error executing %s", cmd)
	}

	for _, tgt := range tgts {
		total[tgt]++
	}
	runAndVerifyProbe(t, p, tgts, total, success)

	// Compute total numbder of event metrics:
	runCnt := 2
	totalEventMetrics := runCnt * len(tgts)
	totalEventMetrics += 3 * len(tgts) // cmd, args, env from successful run
	ems, err := testutils.MetricsFromChannel(p.dataChan, totalEventMetrics, time.Second)
	if err != nil {
		t.Error(err)
	}
	mmap := testutils.MetricsMapByTarget(ems)

	for i, tgt := range tgts {
		// Verify number of values received for each metric
		for cnt, names := range map[int][]string{
			runCnt: {"total", "success", "latency"},
			1:      {"cmd", "args", "env"},
		} {
			for _, name := range names {
				assert.Lenf(t, mmap[tgt][name], cnt, "Wrong number of values for metric (%s) for target (%s)", name, tgt)
			}
		}

		// Verify values for each output metric
		assert.Equal(t, "\""+wantArgs[i]+"\"", mmap[tgt]["args"][0].String(), "Wrong value for args metric")
		assert.Equal(t, "\""+wantEnv+"\"", mmap[tgt]["env"][0].String(), "Wrong value for env metric")
		assert.Equal(t, "\""+wantCmd+"\"", mmap[tgt]["cmd"][0].String(), "Wrong value for cmd metric")
	}
}

func TestProbeOnceMode(t *testing.T) {
	var tests = []struct {
		name     string
		cmd      string
		tgts     []string
		wantArgs []string
	}{
		{
			name:     "no-substitutions",
			cmd:      "/test/cmd --address=a.com --arg2",
			tgts:     []string{"target1", "target2"},
			wantArgs: []string{"--address=a.com,--arg2", "--address=a.com,--arg2"},
		},
		{
			name:     "arg-substitutions",
			cmd:      "/test/cmd --address=@target@ --arg2",
			tgts:     []string{"target1", "target2"},
			wantArgs: []string{"--address=target1,--arg2", "--address=target2,--arg2"},
		},
	}

	for _, tt := range tests {
		envVars := map[string]string{"key": "secret", "client": "client2"}
		wantCmd := "/test/cmd"
		wantEnv := "client=client2 key=secret"
		t.Run(tt.name, func(t *testing.T) {
			testProbeOnceMode(t, tt.cmd, tt.tgts, envVars, wantCmd, wantEnv, tt.wantArgs)
		})
	}
}

func TestUpdateLabelKeys(t *testing.T) {
	c := &configpb.ProbeConf{
		Options: []*configpb.ProbeConf_Option{
			{
				Name:  proto.String("target"),
				Value: proto.String("@target@"),
			},
			{
				Name:  proto.String("probe"),
				Value: proto.String("@probe@"),
			},
		},
	}
	p := &Probe{
		name:    "probeP",
		c:       c,
		cmdArgs: []string{"--server", "@target.label.fqdn@:@port@"},
	}

	p.updateLabelKeys()

	expected := map[string]bool{
		"target":            true,
		"port":              true,
		"probe":             true,
		"target.label.fqdn": true,
	}

	if !reflect.DeepEqual(p.labelKeys, expected) {
		t.Errorf("p.labelKeys got: %v, want: %v", p.labelKeys, expected)
	}

	gotLabels := p.labels(endpoint.Endpoint{
		Name: "targetA",
		Port: 8080,
		Labels: map[string]string{
			"fqdn": "targetA.svc.local",
		},
	})
	wantLabels := map[string]string{
		"target.label.fqdn": "targetA.svc.local",
		"port":              "8080",
		"probe":             "probeP",
		"target":            "targetA",
	}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Errorf("p.labels got: %v, want: %v", gotLabels, wantLabels)
	}
}

// TestSendRequest verifies that sendRequest sends appropriately populated
// ProbeRequest.
func TestSendRequest(t *testing.T) {
	p := &Probe{}
	p.Init("testprobe", &options.Options{
		ProbeConf: &configpb.ProbeConf{
			Options: []*configpb.ProbeConf_Option{
				{
					Name:  proto.String("target"),
					Value: proto.String("@target@"),
				},
			},
			Command: proto.String("./testCommand"),
		},
		Targets: targets.StaticTargets("localhost"),
	})
	var buf bytes.Buffer
	p.cmdStdin = &buf

	requestID := int32(1234)
	target := "localhost"

	err := p.sendRequest(requestID, endpoint.Endpoint{Name: target})
	if err != nil {
		t.Errorf("Failed to sendRequest: %v", err)
	}
	req := new(serverpb.ProbeRequest)
	var length int
	_, err = fmt.Fscanf(&buf, "\nContent-Length: %d\n\n", &length)
	if err != nil {
		t.Errorf("Failed to read header: %v", err)
	}
	err = proto.Unmarshal(buf.Bytes(), req)
	if err != nil {
		t.Fatalf("Failed to Unmarshal probe Request: %v", err)
	}
	if got, want := req.GetRequestId(), requestID; got != requestID {
		t.Errorf("req.GetRequestId() = %q, want %v", got, want)
	}
	opts := req.GetOptions()
	if len(opts) != 1 {
		t.Errorf("req.GetOptions() = %q (%v), want only one item", opts, len(opts))
	}
	if got, want := opts[0].GetName(), "target"; got != want {
		t.Errorf("opts[0].GetName() = %q, want %q", got, want)
	}
	if got, want := opts[0].GetValue(), target; got != target {
		t.Errorf("opts[0].GetValue() = %q, want %q", got, want)
	}
}

func TestUpdateTargets(t *testing.T) {
	p := &Probe{}
	err := p.Init("testprobe", &options.Options{
		ProbeConf: &configpb.ProbeConf{
			Command: proto.String("./testCommand"),
		},
		Targets: targets.StaticTargets("2.2.2.2"),
	})
	if err != nil {
		t.Fatalf("Got error while initializing the probe: %v", err)
	}

	p.updateTargets()
	ep := endpoint.Endpoint{Name: "2.2.2.2"}
	latVal := p.results[ep.Key()].latency
	if _, ok := latVal.(*metrics.Float); !ok {
		t.Errorf("latency value type is not metrics.Float: %v", latVal)
	}

	// Test with latency distribution option set.
	p.opts.LatencyDist = metrics.NewDistribution([]float64{0.1, 0.2, 0.5})
	delete(p.results, ep.Key())
	p.updateTargets()
	latVal = p.results[ep.Key()].latency
	if _, ok := latVal.(*metrics.Distribution); !ok {
		t.Errorf("latency value type is not metrics.Distribution: %v", latVal)
	}
}

func verifyProcessedResult(t *testing.T, p *Probe, r *result, success int64, name string, val int64, extraLabels map[string]string) {
	t.Helper()

	t.Log(val)

	testTarget := "test-target"
	if r.success != success {
		t.Errorf("r.success=%d, expected=%d", r.success, success)
	}

	// Get 2 EventMetrics = default metrics, and payload metrics.
	ems, err := testutils.MetricsFromChannel(p.dataChan, 2, time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}

	mmap, lmap := testutils.MetricsMapByTarget(ems), testutils.LabelsMapByTarget(ems)

	if mmap[testTarget] == nil || lmap[testTarget] == nil {
		t.Fatalf("Payload metrics for target %s missing", testTarget)
	}

	if len(mmap[testTarget][name]) < 1 {
		t.Fatalf("Payload metric %s is missing in %+v", name, mmap[testTarget])
	}

	assert.Equal(t, val, mmap[testTarget][name][0].(metrics.NumValue).Int64(), "metric %s value mismatch", name)

	expectedLabels := map[string]string{"ptype": "external", "probe": "testprobe", "dst": "test-target"}
	for k, v := range extraLabels {
		expectedLabels[k] = v
	}
	assert.Equal(t, expectedLabels, lmap[testTarget], "labels mismatch")
}

func TestProcessProbeResult(t *testing.T) {
	tests := []struct {
		desc             string
		aggregate        bool
		payloads         []string
		additionalLabels map[string]string
		wantValues       []int64
		wantExtraLabels  map[string]string
	}{
		{
			desc:       "with-aggregation-enabled-no-labels",
			aggregate:  true,
			payloads:   []string{"p-failures 14", "p-failures 11"},
			wantValues: []int64{14, 25},
		},
		{
			desc:      "with-aggregation-enabled-with-labels",
			aggregate: true,
			payloads: []string{
				"p-failures{service=serviceA,db=dbA} 14",
				"p-failures{service=serviceA,db=dbA} 11",
			},
			wantValues: []int64{14, 25},
			wantExtraLabels: map[string]string{
				"service": "serviceA",
				"db":      "dbA",
			},
		},
		{
			desc:      "with-aggregation-disabled",
			aggregate: false,
			payloads: []string{
				"p-failures{service=serviceA,db=dbA} 14",
				"p-failures{service=serviceA,db=dbA} 11",
			},
			wantValues: []int64{14, 11},
			wantExtraLabels: map[string]string{
				"service": "serviceA",
				"db":      "dbA",
			},
		},
		{
			desc:      "with-additional-labels",
			aggregate: false,
			payloads: []string{
				"p-failures{service=serviceA,db=dbA} 14",
				"p-failures{service=serviceA,db=dbA} 11",
			},
			additionalLabels: map[string]string{"dc": "xx"},
			wantValues:       []int64{14, 11},
			wantExtraLabels: map[string]string{
				"service": "serviceA",
				"db":      "dbA",
				"dc":      "xx",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := &Probe{}
			opts := options.DefaultOptions()
			opts.ProbeConf = &configpb.ProbeConf{
				OutputMetricsOptions: &payloadconfigpb.OutputMetricsOptions{
					AggregateInCloudprober: proto.Bool(test.aggregate),
				},
				Command: proto.String("./testCommand"),
			}
			for k, v := range test.additionalLabels {
				opts.AdditionalLabels = append(opts.AdditionalLabels, options.ParseAdditionalLabel(&probeconfigpb.AdditionalLabel{
					Key:   proto.String(k),
					Value: proto.String(v),
				}))
			}
			err := p.Init("testprobe", opts)
			if err != nil {
				t.Fatal(err)
			}

			p.dataChan = make(chan *metrics.EventMetrics, 20)

			r := &result{
				latency: metrics.NewFloat(0),
			}

			// First run
			p.processProbeResult(&probeStatus{
				target:  endpoint.Endpoint{Name: "test-target"},
				success: true,
				latency: time.Millisecond,
				payload: test.payloads[0],
			}, r)

			wantSuccess := int64(1)
			verifyProcessedResult(t, p, r, wantSuccess, "p-failures", test.wantValues[0], test.wantExtraLabels)

			// Second run
			p.processProbeResult(&probeStatus{
				target:  endpoint.Endpoint{Name: "test-target"},
				success: true,
				latency: time.Millisecond,
				payload: test.payloads[1],
			}, r)
			wantSuccess++

			verifyProcessedResult(t, p, r, wantSuccess, "p-failures", test.wantValues[1], test.wantExtraLabels)
		})
	}
}

func TestCommandParsing(t *testing.T) {
	p := createTestProbe("./test-command --flag1 one --flag23 \"two three\"", nil)

	wantCmdName := "./test-command"
	if p.cmdName != wantCmdName {
		t.Errorf("Got command name=%s, want command name=%s", p.cmdName, wantCmdName)
	}

	wantArgs := []string{"--flag1", "one", "--flag23", "two three"}
	if !reflect.DeepEqual(p.cmdArgs, wantArgs) {
		t.Errorf("Got command args=%v, want command args=%v", p.cmdArgs, wantArgs)
	}
}

type fakeCommand struct {
	exitCtx  context.Context
	startCtx context.Context
	waitErr  error
}

func (fc *fakeCommand) Wait() error {
	select {
	case <-fc.exitCtx.Done():
	case <-fc.startCtx.Done():
	}
	return fc.waitErr
}

func TestMonitorCommand(t *testing.T) {
	tests := []struct {
		desc       string
		waitErr    error
		finishCmd  bool
		cancelCtx  bool
		wantErr    bool
		wantStderr bool
	}{
		{
			desc:      "Command exit with no error",
			finishCmd: true,
			wantErr:   false,
		},
		{
			desc:      "Cancel context, no error",
			cancelCtx: true,
			wantErr:   false,
		},
		{
			desc:       "command exit with exit error",
			finishCmd:  true,
			waitErr:    &exec.ExitError{Stderr: []byte("exit-error exiting")},
			wantErr:    true,
			wantStderr: true,
		},
		{
			desc:       "command exit with no exit error",
			finishCmd:  true,
			waitErr:    errors.New("some-error"),
			wantErr:    true,
			wantStderr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			exitCtx, exitFunc := context.WithCancel(context.Background())
			startCtx, startCancelFunc := context.WithCancel(context.Background())
			cmd := &fakeCommand{
				exitCtx:  exitCtx,
				startCtx: startCtx,
				waitErr:  test.waitErr,
			}

			p := &Probe{}
			errCh := make(chan error)
			go func() {
				errCh <- p.monitorCommand(startCtx, cmd)
			}()

			if test.finishCmd {
				exitFunc()
			}
			if test.cancelCtx {
				startCancelFunc()
			}

			err := <-errCh
			if (err != nil) != test.wantErr {
				t.Errorf("Got error: %v, want error?= %v", err, test.wantErr)
			}

			if err != nil {
				if test.wantStderr && !strings.Contains(err.Error(), "Stderr") {
					t.Errorf("Want std err: %v, got std err: %v", test.wantStderr, strings.Contains(err.Error(), "Stderr"))
				}
			}

			exitFunc()
			startCancelFunc()
		})
	}
}

// TestShellProcessSuccess is just a helper function that we use to test the
// actual command execution (from startCmdIfNotRunning, e.g.).
func TestShellProcessSuccess(t *testing.T) {
	// Ignore this test if it's not being run as a subprocess for another test.
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	pauseSec, err := strconv.Atoi(os.Getenv("PAUSE"))
	if err != nil {
		t.Fatal(err)
	}
	pauseTime := time.Duration(pauseSec) * time.Second
	time.Sleep(pauseTime)
	os.Exit(0)
}

func TestProbeStartCmdIfNotRunning(t *testing.T) {
	isCmdRunning := func(p *Probe) bool {
		p.cmdRunningMu.Lock()
		defer p.cmdRunningMu.Unlock()
		return p.cmdRunning
	}

	tests := []struct {
		pauseSec int
		wantErr  bool
	}{
		{
			pauseSec: 0,
		},
		{
			pauseSec: 60,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("pause=%d", test.pauseSec), func(t *testing.T) {
			// testCommand is just a placeholder here. We replace cmdName and
			// cmdArgs after creating the probe.
			p := createTestProbe("/testCommand", map[string]string{
				"GO_TEST_PROCESS": "1",
				"PAUSE":           strconv.Itoa(test.pauseSec),
			})

			p.cmdName = os.Args[0]
			p.cmdArgs = []string{"-test.run=TestShellProcessSuccess"}

			if err := p.startCmdIfNotRunning(context.Background()); err != nil {
				t.Fatal(err)
			}

			stdin, stdout, stderr := p.cmdStdin, p.cmdStdout, p.cmdStderr
			// Wait for the command to finish if it's not supposed to wait
			if test.pauseSec == 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				for {
					select {
					case <-ctx.Done():
						t.Fatal("Timed out waiting for command to finish")
					default:
					}
					if !isCmdRunning(p) {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}

			if err := p.startCmdIfNotRunning(context.Background()); err != nil {
				t.Fatal(err)
			}

			// stdin, stdout, and stdout should get reset if pausesec is 0.
			resetOrNot := test.pauseSec == 0
			assert.Equal(t, resetOrNot, stdin != p.cmdStdin)
			assert.Equal(t, resetOrNot, stdout != p.cmdStdout)
			assert.Equal(t, resetOrNot, stderr != p.cmdStderr)
		})
	}
}
