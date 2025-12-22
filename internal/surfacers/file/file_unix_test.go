// Copyright 2025 The Cloudprober Authors.
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

//go:build !windows

package file

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/file/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
	"google.golang.org/protobuf/proto"
)

func TestWriteChannelFull(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "file_test")
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

	// This write should fail as the channel is full.
	s.Write(context.Background(), em)

	if !strings.Contains(buf.String(), "write channel is full") {
		t.Errorf("Expected log on write channel full. Got: %s", buf.String())
	}

	wg.Done()
}
