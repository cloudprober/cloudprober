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
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// maxQueuedLogBytes is the most log data we hold in memory while waiting for
// stderr to accept it.
const maxQueuedLogBytes = 4 << 20

// asyncWriter is an io.Writer that never blocks the caller. Writes are queued
// and written to the underlying writer, in order, by a single goroutine.
//
// Writes to stderr can block indefinitely if whatever reads the other end of
// the pipe stalls (e.g. the container runtime's log writer on a node with a
// stuck disk). Without this, every goroutine that logs, including probe
// goroutines, would block behind that write. With it, only the drain
// goroutine blocks; once the queue fills up, new entries are dropped and
// counted, and a notice marking the gap precedes the next accepted entry.
type asyncWriter struct {
	w        io.Writer
	maxBytes int

	mu     sync.Mutex
	queue  [][]byte
	queued int // Bytes queued or being written.

	ready   chan struct{} // Wakes up the drain goroutine.
	dropped atomic.Int64  // Total entries dropped.
	noted   int64         // Drops already covered by a notice.
}

func newAsyncWriter(w io.Writer, maxBytes int) *asyncWriter {
	aw := &asyncWriter{
		w:        w,
		maxBytes: maxBytes,
		ready:    make(chan struct{}, 1),
	}
	go aw.drain()
	return aw
}

// Write queues p for writing and always returns len(p), nil. If the queue is
// full, p is dropped.
func (aw *asyncWriter) Write(p []byte) (int, error) {
	aw.mu.Lock()
	if aw.queued+len(p) > aw.maxBytes {
		aw.dropped.Add(1)
		aw.mu.Unlock()
		return len(p), nil
	}
	// First entry accepted after drops: put a notice where the gap is.
	if d := aw.dropped.Load(); d > aw.noted {
		notice := dropNotice(d - aw.noted)
		aw.queue = append(aw.queue, notice)
		aw.queued += len(notice)
		aw.noted = d
	}
	// Copy p: slog handlers reuse their buffers after Write returns.
	aw.queue = append(aw.queue, bytes.Clone(p))
	aw.queued += len(p)
	aw.mu.Unlock()

	select {
	case aw.ready <- struct{}{}:
	default:
	}
	return len(p), nil
}

func (aw *asyncWriter) drain() {
	for range aw.ready {
		aw.mu.Lock()
		batch := aw.queue
		aw.queue = nil
		aw.mu.Unlock()

		for _, b := range batch {
			aw.w.Write(b)
			aw.mu.Lock()
			aw.queued -= len(b)
			aw.mu.Unlock()
		}
	}
}

func dropNotice(n int64) []byte {
	var buf bytes.Buffer
	r := slog.NewRecord(time.Now(), slog.LevelWarn, "Dropped log entries because writes to stderr were blocked", 0)
	r.AddAttrs(slog.String("system", defaultSystemName), slog.Int64("dropped_entries", n))
	slogHandler(&buf).Handle(context.Background(), r)
	return buf.Bytes()
}

// Flush waits until all queued entries have been written, or until timeout
// expires. It reports whether the queue was fully written.
func (aw *asyncWriter) Flush(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		aw.mu.Lock()
		queued := aw.queued
		aw.mu.Unlock()

		if queued == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}
