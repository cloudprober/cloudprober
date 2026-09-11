// Copyright 2026 The Cloudprober Authors.
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

package logger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// blockingWriter blocks all writes until unblock is closed, like a stderr
// pipe whose reader has stalled.
type blockingWriter struct {
	unblock chan struct{}

	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	<-w.unblock
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *blockingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func entry(i int) string {
	return fmt.Sprintf("entry-%02d\n", i) // 9 bytes
}

func TestAsyncWriterBlockedWriter(t *testing.T) {
	bw := &blockingWriter{unblock: make(chan struct{})}
	aw := newAsyncWriter(bw, 10*len(entry(0)))

	// Room for 10 entries; the remaining 5 should be dropped, without
	// blocking.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 15 {
			n, err := aw.Write([]byte(entry(i)))
			assert.NoError(t, err)
			assert.Equal(t, len(entry(i)), n)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Write blocked on a blocked writer")
	}

	assert.Equal(t, int64(5), aw.dropped.Load())
	assert.False(t, aw.Wait(0), "Wait(0) with a blocked writer")
	assert.False(t, aw.Wait(50*time.Millisecond), "Wait with a blocked writer")

	close(bw.unblock)
	assert.True(t, aw.Wait(5*time.Second), "Wait after unblocking")
	assert.True(t, aw.Wait(0), "Wait(0) with nothing queued")

	// The next accepted entry is preceded by a notice about the drops.
	aw.Write([]byte(entry(15)))
	assert.True(t, aw.Wait(5*time.Second))

	lines := strings.Split(strings.TrimSuffix(bw.String(), "\n"), "\n")
	if !assert.Len(t, lines, 12) {
		return
	}
	for i := range 10 {
		assert.Equal(t, strings.TrimSuffix(entry(i), "\n"), lines[i])
	}
	assert.Contains(t, lines[10], "level=WARN")
	assert.Contains(t, lines[10], "dropped_entries=5")
	assert.Equal(t, strings.TrimSuffix(entry(15), "\n"), lines[11])
}

func TestAsyncWriterCopiesInput(t *testing.T) {
	bw := &blockingWriter{unblock: make(chan struct{})}
	aw := newAsyncWriter(bw, 1024)

	p := []byte(entry(0))
	aw.Write(p)
	copy(p, entry(1)) // Caller reuses its buffer, as slog does.

	close(bw.unblock)
	assert.True(t, aw.Wait(5*time.Second))
	assert.Equal(t, entry(0), bw.String())
}

func TestAsyncWriterConcurrentWrites(t *testing.T) {
	bw := &blockingWriter{unblock: make(chan struct{})}
	close(bw.unblock)
	aw := newAsyncWriter(bw, maxQueuedLogBytes)

	var wg sync.WaitGroup
	for g := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				aw.Write([]byte(fmt.Sprintf("g%d-%d\n", g, i)))
			}
		}()
	}
	wg.Wait()

	assert.True(t, aw.Wait(5*time.Second))
	assert.Equal(t, 1000, strings.Count(bw.String(), "\n"))
	assert.Equal(t, int64(0), aw.dropped.Load())
}

// Observing the stderr writer must work without one: starting it just to
// answer a question would start its drain goroutine in processes that never
// log to stderr.
func TestObserversWithoutWriter(t *testing.T) {
	assert.Zero(t, droppedEntries(nil), "droppedEntries(nil)")
	assert.True(t, waitForStderr(nil, 0), "waitForStderr(nil)")
}

func TestObserversWithWriter(t *testing.T) {
	bw := &blockingWriter{unblock: make(chan struct{})}
	aw := newAsyncWriter(bw, len(entry(0))) // Room for one entry.

	aw.Write([]byte(entry(0))) // Queued, and then blocked in the writer.
	aw.Write([]byte(entry(1))) // Dropped: no room left.

	assert.Equal(t, int64(1), droppedEntries(aw))
	assert.False(t, waitForStderr(aw, 0), "waitForStderr with a blocked writer")

	close(bw.unblock)
	assert.True(t, waitForStderr(aw, 5*time.Second), "waitForStderr after unblocking")
	assert.Equal(t, entry(0), bw.String())
}

func TestStderrWriter(t *testing.T) {
	w := Stderr()
	if w == nil {
		t.Fatal("Stderr() = nil, want the async stderr writer")
	}
	assert.True(t, w == Stderr(), "Stderr() returned a different writer")

	// The exported observers read the writer Stderr() returns, which the
	// call above has certainly started by now.
	assert.Equal(t, asyncStderr().dropped.Load(), DroppedEntries())
	WaitForStderr(time.Second)
}
