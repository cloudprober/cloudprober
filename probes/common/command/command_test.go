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

package command

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

	pauseTime := 10 * time.Millisecond
	pauseSec, _ := strconv.Atoi(os.Getenv("GO_CP_TEST_PAUSE"))
	if pauseSec != 0 {
		pauseTime = time.Duration(pauseSec) * time.Second
	}

	if len(os.Args) > 4 {
		fmt.Printf("cmd \"%s\"\n", os.Args[3])
		time.Sleep(pauseTime)
		fmt.Printf("args \"%s\"\n", strings.Join(os.Args[4:], ","))
		fmt.Printf("env \"%s\"\n", strings.Join(exportEnvList, ","))
	}

	if os.Getenv("GO_CP_TEST_PROCESS_FAIL") != "" {
		os.Exit(1)
	}
	os.Exit(0)
}

func testCommandExecute(t *testing.T, disableStreaming bool) {
	t.Helper()

	// This is the base command, it gets appended to the test command.
	cmd := []string{"/test/cmd", "--address=a.com", "--arg2"}

	p := &Command{
		CmdLine: append([]string{os.Args[0], "-test.run=TestShellProcessSuccess", "--", cmd[0]}, cmd[1:]...), // prepend os.Args[0],
		EnvVars: []string{
			"GO_CP_TEST_PROCESS=1",
			"GO_CP_TEST_PIDS_FILE=pidsFile",
		},
	}

	// Output by TestShellProcessSuccess from the first run
	wantOutput := []string{
		"cmd \"/test/cmd\"",
		"args \"--address=a.com,--arg2\"",
		"env \"GO_CP_TEST_PROCESS=1,GO_CP_TEST_PIDS_FILE=pidsFile\"",
	}

	var output []string
	var outputTS []int64
	var outputMu sync.RWMutex
	if !disableStreaming {
		p.ProcessStreamingOutput = func(line []byte) {
			outputMu.Lock()
			defer outputMu.Unlock()
			output = append(output, string(line))
			outputTS = append(outputTS, time.Now().UnixMilli())
		}
	}

	t.Run("first-run", func(t *testing.T) {
		payload, err := p.Execute(context.Background(), nil)
		assert.NoError(t, err)

		if !disableStreaming {
			assert.Equal(t, "", payload)
		} else {
			assert.Equal(t, strings.Join(wantOutput, "\n")+"\n", payload)
		}
	})

	wantOutput = append(wantOutput, wantOutput...)
	wantOutput[len(wantOutput)-1] = "env \"GO_CP_TEST_PROCESS_FAIL=1,GO_CP_TEST_PROCESS=1,GO_CP_TEST_PIDS_FILE=pidsFile\""
	t.Run("second-run", func(t *testing.T) {
		os.Setenv("GO_CP_TEST_PROCESS_FAIL", "1")
		defer os.Unsetenv("GO_CP_TEST_PROCESS_FAIL")

		payload, err := p.Execute(context.Background(), nil)
		assert.Error(t, err)
		assert.Equal(t, "", payload)
	})

	// Make process timeout between "cmd" and "args" output
	wantOutput = append(wantOutput, wantOutput[0])
	t.Run("third-run-with-timeout", func(t *testing.T) {
		os.Setenv("GO_CP_TEST_PAUSE", "1")
		defer os.Unsetenv("GO_CP_TEST_PAUSE")

		// Give process enough time to get the first line
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		payload, err := p.Execute(ctx, nil)
		assert.Error(t, err)
		assert.Equal(t, "", payload)
	})

	if !disableStreaming {
		outputMu.RLock()
		defer outputMu.RUnlock()
		assert.Equal(t, wantOutput, output)

		// Verify that difference between first two timestamps is at least 10ms
		assert.True(t, outputTS[1] >= outputTS[0]+10, "gap between cmd and arg not more than 10ms")
		// Verify that difference between second two timestamps is less than 10ms
		assert.True(t, outputTS[2] < outputTS[1]+10, "gap between arg and env not less than 10ms")
	}
}

func TestCommand(t *testing.T) {
	for _, disableStreaming := range []bool{false, true} {
		t.Run(fmt.Sprintf("disable-streaming:%v", disableStreaming), func(t *testing.T) {
			testCommandExecute(t, disableStreaming)
		})
	}
}
