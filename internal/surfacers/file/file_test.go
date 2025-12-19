// Copyright 2017 The Cloudprober Authors.
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

package file

/*
These tests wait 100 milliseconds in between a write command and the read of the
file that it writes to to allow for the write to disk to complete. This could
lead to flakiness in the future.
*/

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/protobuf/proto"

	"github.com/cloudprober/cloudprober/internal/surfacers/common/compress"
	configpb "github.com/cloudprober/cloudprober/internal/surfacers/file/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
)

func TestWrite(t *testing.T) {
	testWrite(t, false)
}

func TestWriteWithCompression(t *testing.T) {
	testWrite(t, true)
}

func testWrite(t *testing.T, compressionEnabled bool) {
	t.Helper()

	tests := []struct {
		description string
		em          *metrics.EventMetrics
	}{
		{
			description: "file write of float data",
			em:          metrics.NewEventMetrics(time.Now()).AddMetric("float-test", metrics.NewInt(123456)),
		},
		{
			description: "file write of string data",
			em:          metrics.NewEventMetrics(time.Now()).AddMetric("string-test", metrics.NewString("test-string")),
		},
	}

	for _, tt := range tests {

		// Create temporary file so we don't accidentally overwrite
		// another file during test.
		f, err := ioutil.TempFile("", "file_test")
		if err != nil {
			t.Errorf("Unable to create a new file for testing: %v\ntest description: %s", err, tt.description)
		}
		defer os.Remove(f.Name())

		s := &Surfacer{
			c: &configpb.SurfacerConf{
				FilePath:           proto.String(f.Name()),
				CompressionEnabled: proto.Bool(compressionEnabled),
			},
			opts: &options.Options{
				MetricsBufferSize: 1000,
			},
		}
		id := time.Now().UnixNano()
		err = s.init(context.Background(), id)
		if err != nil {
			t.Errorf("Unable to create a new file surfacer: %v\ntest description: %s", err, tt.description)
		}

		s.Write(context.Background(), tt.em)
		s.close()

		dat, err := os.ReadFile(f.Name())
		if err != nil {
			t.Errorf("Unable to open test output file for reading: %v\ntest description: %s", err, tt.description)
		}

		expectedStr := fmt.Sprintf("%s %d %s\n", s.c.GetPrefix(), id, tt.em.String())
		if compressionEnabled {
			compressed, err := compress.Compress([]byte(expectedStr))
			if err != nil {
				t.Errorf("Unexpected error while compressing bytes: %v, Input: %s", err, expectedStr)
			}
			expectedStr = string(compressed) + "\n"
		}

		if diff := pretty.Compare(expectedStr, string(dat)); diff != "" {
			t.Errorf("Message written does not match expected output (-want +got):\n%s\ntest description: %s", diff, tt.description)
		}
	}
}

func TestWriteChannelFull(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "file_test")
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	pipeName := tmpdir + "/test-pipe"
	if err := syscall.Mkfifo(pipeName, 0666); err != nil {
		t.Fatalf("Error creating named pipe: %v", err)
	}

	// This goroutine will open the pipe for reading, but will not read from it.
	// This will cause the writer to block, and the channel to fill up.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f, err := os.OpenFile(pipeName, os.O_RDONLY, 0666)
		if err != nil {
			t.Errorf("Error opening pipe for reading: %v", err)
			return
		}
		// Block until the test is over.
		time.Sleep(2 * time.Second)
		f.Close()
	}()

	var buf bytes.Buffer
	l := logger.New(logger.WithWriter(&buf))

	s, err := New(context.Background(), &configpb.SurfacerConf{
		FilePath: proto.String(pipeName),
	}, &options.Options{
		MetricsBufferSize: 1,
	}, l)
	if err != nil {
		t.Fatalf("Unable to create a new file surfacer: %v", err)
	}
	defer s.close()

	em := metrics.NewEventMetrics(time.Now()).AddMetric("test", metrics.NewInt(1))

	// First write should succeed and fill the channel.
	s.Write(context.Background(), em)

	// Second write should fail as the channel is full.
	s.Write(context.Background(), em)

	if !strings.Contains(buf.String(), "write channel is full") {
		t.Errorf("Expected log on write channel full. Got: %s", buf.String())
	}

	wg.Done()
}
