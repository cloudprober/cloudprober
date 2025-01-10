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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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
	"github.com/cloudprober/cloudprober/probes/options"
	probeconfigpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var pidsFile string

func setProbeOptions(p *Probe, name, value string) {
	for _, opt := range p.c.Options {
		if opt.GetName() == name {
			opt.Value = proto.String(value)
			break
		}
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

func createTestProbe(cmd string, envVar map[string]string, mode configpb.ProbeConf_Mode) *Probe {
	probeConf := &configpb.ProbeConf{
		Mode: mode.Enum(),
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
		LatencyMetricName: "latency",
	})

	return p
}

// TestShellProcessSuccess is just a helper function that we use to test the
// actual command execution (from startCmdIfNotRunning, e.g.).
func TestShellProcessSuccess(t *testing.T) {
	// Ignore this test if it's not being run as a subprocess for another test.
	if os.Getenv("GO_CP_TEST_PROCESS") != "1" {
		return
	}

	exportEnvList := []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GO_CP_TEST") {
			exportEnvList = append(exportEnvList, env)
		}
	}
	fmt.Fprintf(os.Stderr, "Running test command. Env: %s\n", strings.Join(exportEnvList, ","))

	if len(os.Args) > 4 {
		fmt.Printf("cmd \"%s\"\n", os.Args[3])
		time.Sleep(10 * time.Millisecond)
		fmt.Printf("args \"%s\"\n", strings.Join(os.Args[4:], ","))
		fmt.Printf("env \"%s\"\n", strings.Join(exportEnvList, ","))
	}

	if os.Getenv("GO_CP_TEST_PROCESS_FAIL") != "" {
		os.Exit(1)
	}

	pauseSec, err := strconv.Atoi(os.Getenv("GO_CP_TEST_PAUSE"))
	if err != nil {
		pauseSec = 0
	}
	if pauseSec != 0 {
		fmt.Fprintf(os.Stderr, "Going to sleep for %d\n", pauseSec)
		time.Sleep(time.Duration(pauseSec) * time.Second)
	}
	os.Exit(0)
}

func testProbeOnceMode(t *testing.T, cmd string, tgts []string, runTwice, disableStreaming bool, wantCmd string, wantArgs []string) {
	t.Helper()

	p := createTestProbe(cmd, map[string]string{
		"GO_CP_TEST_PROCESS":   "1",
		"GO_CP_TEST_PIDS_FILE": pidsFile,
	}, configpb.ProbeConf_ONCE)

	p.cmdArgs = append([]string{"-test.run=TestShellProcessSuccess", "--", p.cmdName}, p.cmdArgs...)
	p.cmdName = os.Args[0]
	p.c.Mode = configpb.ProbeConf_ONCE.Enum()
	p.c.DisableStreamingOutputMetrics = proto.Bool(disableStreaming)
	// We don't rely on timeout but give process enough time to finish,
	// especially for CI.
	p.opts.Timeout = 10 * time.Second

	total, success := make(map[string]int64), make(map[string]int64)
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}

	var standardEMCount, outputEMCount int
	numOutVars := 3

	t.Run("first-run", func(t *testing.T) {
		standardEMCount += len(tgts)            // 1 EM for each target
		outputEMCount += numOutVars * len(tgts) // 3 EMs for each target
		runAndVerifyProbe(t, p, tgts, total, success)
	})

	if runTwice {
		t.Run("second-run", func(t *testing.T) {
			// Cause our test command to fail immediately
			os.Setenv("GO_CP_TEST_PROCESS_FAIL", "1")
			defer os.Unsetenv("GO_CP_TEST_PROCESS_FAIL")

			standardEMCount += len(tgts)            // 1 EM for each target
			outputEMCount += numOutVars * len(tgts) // 3 EMs for each target
			for _, tgt := range tgts {
				total[tgt]++
			}
			runAndVerifyProbe(t, p, tgts, total, success)
		})
	}

	ems, err := testutils.MetricsFromChannel(p.dataChan, standardEMCount+outputEMCount, time.Second)
	if err != nil {
		t.Error(err)
	}

	mmap := testutils.MetricsMapByTarget(ems)
	emMap := testutils.EventMetricsByTargetMetric(ems)

	for i, tgt := range tgts {
		// Verify number of values received for each metric
		for names, wantCount := range map[[3]string]int{
			{"total", "success", "latency"}: standardEMCount / len(tgts),
			{"cmd", "args", "env"}:          outputEMCount / (len(tgts) * numOutVars),
		} {
			for _, name := range names {
				assert.Lenf(t, mmap[tgt][name], wantCount, "Wrong number of values for metric (%s) for target (%s)", name, tgt)
			}
		}

		wantEnv := [][]string{{"GO_CP_TEST_PROCESS=1"}, {"GO_CP_TEST_PROCESS_FAIL=1", "GO_CP_TEST_PROCESS=1"}}

		// Verify values for each output metric
		for j := range mmap[tgt]["args"] {
			assert.Equalf(t, "\""+wantArgs[i]+"\"", mmap[tgt]["args"][j].String(), "Wrong value for args[%d] metric for target (%s)", j, tgt)
			assert.Equalf(t, "\""+wantCmd+"\"", mmap[tgt]["cmd"][j].String(), "Wrong value for cmd[%d] metric for target (%s)", j, tgt)

			for _, e := range wantEnv[j] {
				assert.Contains(t, mmap[tgt]["env"][j].String(), e, "env[%d] metric for target (%s) doesn't contain expected value", j, tgt)
			}
		}

		cmdTime := emMap[tgt]["cmd"][0].Timestamp
		argsTime := emMap[tgt]["args"][0].Timestamp
		if disableStreaming {
			assert.Equal(t, cmdTime, argsTime, "cmd and args metrics should have same timestamp")
		} else {
			assert.True(t, cmdTime.Before(argsTime), "cmd metric should have earlier timestamp than args metric")
		}
	}
}

func TestProbeOnceMode(t *testing.T) {
	var tests = []struct {
		name             string
		cmd              string
		tgts             []string
		runTwice         bool
		disableStreaming bool
		wantArgs         []string
	}{
		{
			name:     "no-substitutions",
			cmd:      "/test/cmd --address=a.com --arg2",
			tgts:     []string{"target1", "target2"},
			runTwice: true,
			wantArgs: []string{"--address=a.com,--arg2", "--address=a.com,--arg2"},
		},
		{
			name:     "arg-substitutions",
			cmd:      "/test/cmd --address=@target@ --arg2",
			tgts:     []string{"target1", "target2"},
			wantArgs: []string{"--address=target1,--arg2", "--address=target2,--arg2"},
		},
		{
			name:             "streaming-disabled",
			cmd:              "/test/cmd --address=a.com --arg2",
			tgts:             []string{"target1", "target2"},
			disableStreaming: true,
			wantArgs:         []string{"--address=a.com,--arg2", "--address=a.com,--arg2"},
		},
	}

	for _, tt := range tests {
		wantCmd := "/test/cmd"
		t.Run(tt.name, func(t *testing.T) {
			testProbeOnceMode(t, tt.cmd, tt.tgts, tt.runTwice, tt.disableStreaming, wantCmd, tt.wantArgs)
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
				Name:  proto.String("target_name"),
				Value: proto.String("@target.name@"),
			},
			{
				Name:  proto.String("probe"),
				Value: proto.String("@probe@"),
			},
			{
				Name:  proto.String("target_ip"),
				Value: proto.String("@target.ip@"),
			},
			{
				Name:  proto.String("target_port"),
				Value: proto.String("@target.port@"),
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
		"target.name":       true,
		"target.ip":         true,
		"target.port":       true,
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
		IP:   net.ParseIP("127.0.0.1"),
		Labels: map[string]string{
			"fqdn": "targetA.svc.local",
		},
	})
	wantLabels := map[string]string{
		"target.label.fqdn": "targetA.svc.local",
		"port":              "8080",
		"probe":             "probeP",
		"target":            "targetA",
		"target.name":       "targetA",
		"target.ip":         "127.0.0.1",
		"target.port":       "8080",
	}
	assert.Equal(t, wantLabels, gotLabels, "p.labels")
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
	req := new(configpb.ProbeRequest)
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
				success: true,
				latency: time.Millisecond,
				payload: test.payloads[0],
			}, endpoint.Endpoint{Name: "test-target"}, r)

			wantSuccess := int64(1)
			verifyProcessedResult(t, p, r, wantSuccess, "p-failures", test.wantValues[0], test.wantExtraLabels)

			// Second run
			p.processProbeResult(&probeStatus{
				success: true,
				latency: time.Millisecond,
				payload: test.payloads[1],
			}, endpoint.Endpoint{Name: "test-target"}, r)
			wantSuccess++

			verifyProcessedResult(t, p, r, wantSuccess, "p-failures", test.wantValues[1], test.wantExtraLabels)
		})
	}
}

func TestCommandParsing(t *testing.T) {
	p := createTestProbe("./test-command --flag1 one --flag23 \"two three\"", nil, configpb.ProbeConf_ONCE)

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

func TestProbeStartCmdIfNotRunning(t *testing.T) {
	isCmdRunning := func(p *Probe) bool {
		p.cmdRunningMu.Lock()
		defer p.cmdRunningMu.Unlock()
		return p.cmdRunning
	}

	waitForCmdToEnd := func(p *Probe) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	tests := []struct {
		pauseSec int
		wantErr  bool
	}{
		{
			pauseSec: 0,
		},
		{
			pauseSec: 10,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("pause=%d", test.pauseSec), func(t *testing.T) {
			// testCommand is just a placeholder here. We replace cmdName and
			// cmdArgs after creating the probe.
			p := createTestProbe("/testCommand", map[string]string{
				"GO_CP_TEST_PROCESS": "1",
				"GO_CP_TEST_PAUSE":   strconv.Itoa(test.pauseSec),
			}, configpb.ProbeConf_ONCE)

			p.cmdName = os.Args[0]
			p.cmdArgs = []string{"-test.run=TestShellProcessSuccess"}

			if err := p.startCmdIfNotRunning(context.Background()); err != nil {
				t.Fatal(err)
			}

			stdin, stdout, stderr := p.cmdStdin, p.cmdStdout, p.cmdStderr
			// Wait for the command to finish if it's not supposed to wait
			if test.pauseSec == 0 {
				waitForCmdToEnd(p)
			}

			if err := p.startCmdIfNotRunning(context.Background()); err != nil {
				t.Fatal(err)
			}

			// if original process has died, i.e pauseSec == 0, stdin, stdout,
			// and stdout should be different now as new process will have started.
			changedOrNot := test.pauseSec == 0
			assert.Equal(t, changedOrNot, stdin != p.cmdStdin)
			assert.Equal(t, changedOrNot, stdout != p.cmdStdout)
			assert.Equal(t, changedOrNot, stderr != p.cmdStderr)
		})
	}
}

func TestMain(m *testing.M) {
	// In the main process, create temp file to store pids of the forked
	// processes.
	if os.Getenv("GO_CP_TEST_PROCESS") == "" {
		tempFile, err := os.CreateTemp("", "cloudprober-external-test-pids")
		if err != nil {
			log.Fatalf("Failed to create temp file for pids: %v", err)
		}
		pidsFile = tempFile.Name()
		tempFile.Close()
	} else {
		pidsF, err := os.OpenFile(os.Getenv("GO_CP_TEST_PIDS_FILE"), os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open temp file for pids: %v", err)
		}
		pidsF.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
		pidsF.Close()
	}

	status := m.Run()

	// Read pid from pidsFile
	pidsBytes, err := os.ReadFile(pidsFile)
	if err != nil {
		log.Fatalf("Failed to open temp file for pids: %v", err)
	}
	for _, pid := range strings.Fields(strings.TrimSpace(string(pidsBytes))) {
		pid, err := strconv.Atoi(pid)
		if err != nil {
			log.Fatalf("Failed to convert pid (%v) to int, err: %v", pid, err)
		}
		log.Println("Killing pid", pid)
		p, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		p.Kill()
	}

	os.Exit(status)
}

func TestProbeInit(t *testing.T) {
	var tests = []struct {
		name      string
		opts      *options.Options
		wantErr   bool
		wantProbe *Probe
	}{
		{
			name: "no-command",
			opts: &options.Options{
				Targets: targets.StaticTargets("localhost"),
			},
			wantErr: true,
		},
		{
			name: "valid-command",
			opts: &options.Options{
				ProbeConf: &configpb.ProbeConf{
					Command: proto.String("./testCommand --flag1 one --flag23 \"two three\""),
				},
			},
			wantErr: false,
			wantProbe: &Probe{
				cmdName: "./testCommand",
				cmdArgs: []string{"--flag1", "one", "--flag23", "two three"},
			},
		},
		{
			name: "valid-command-server-mode",
			opts: &options.Options{
				ProbeConf: &configpb.ProbeConf{
					Command: proto.String("./testCommand --flag1 one --flag23 \"two three\""),
					Mode:    configpb.ProbeConf_SERVER.Enum(),
				},
			},
			wantErr: false,
			wantProbe: &Probe{
				cmdName: "./testCommand",
				cmdArgs: []string{"--flag1", "one", "--flag23", "two three"},
			},
		},
		{
			name: "bad-command",
			opts: &options.Options{
				ProbeConf: &configpb.ProbeConf{
					Command: proto.String("./testCommand --flag1 one --flag23 \"two three"),
				},
			},
			wantErr: true,
		},
		{
			name: "output-metrics-disabled",
			opts: &options.Options{
				ProbeConf: &configpb.ProbeConf{
					Command:         proto.String("./testCommand --flag1 one --flag23 \"two three\""),
					OutputAsMetrics: proto.Bool(false),
				},
			},
			wantErr: false,
			wantProbe: &Probe{
				cmdName: "./testCommand",
				cmdArgs: []string{"--flag1", "one", "--flag23", "two three"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{}
			err := p.Init("testprobe", tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if tt.wantProbe == nil {
				return
			}
			wantPayloadParser := tt.opts.ProbeConf.(*configpb.ProbeConf).GetOutputAsMetrics()
			assert.Equal(t, tt.wantProbe.cmdName, p.cmdName)
			assert.Equal(t, tt.wantProbe.cmdArgs, p.cmdArgs)
			assert.Equal(t, wantPayloadParser, p.payloadParser != nil)
		})
	}

}
