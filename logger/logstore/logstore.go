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

// Package logstore provides an in-memory log store with per-probe ring buffers
// and a configurable memory ceiling.
package logstore

import (
	"log/slog"
	"slices"
	"sync"
	"time"
)

// LogEntry is a single stored log record.
type LogEntry struct {
	Timestamp time.Time
	Level     slog.Level
	Message   string
	Attrs     map[string]string
	size      int // estimated memory footprint in bytes
}

// estimateSize returns the estimated memory footprint of the entry.
func estimateSize(msg string, attrs map[string]string) int {
	// Base overhead for the struct itself.
	size := 128

	size += len(msg)
	for k, v := range attrs {
		size += len(k) + len(v) + 16 // 16 bytes map entry overhead
	}
	return size
}

// ringBuffer is a dynamically growing circular buffer capped by the LogStore's
// memory ceiling.
type ringBuffer struct {
	entries  []LogEntry
	writeIdx int
	count    int
	memUsed  int64
}

func (rb *ringBuffer) write(entry LogEntry) (evictedSize int64) {
	if rb.count < len(rb.entries) {
		// Still have room in the allocated slice.
		rb.entries[rb.writeIdx] = entry
		rb.writeIdx = (rb.writeIdx + 1) % len(rb.entries)
		rb.count++
		rb.memUsed += int64(entry.size)
		return 0
	}

	// Buffer is full — overwrite oldest entry.
	evicted := rb.entries[rb.writeIdx]
	evictedSize = int64(evicted.size)
	rb.entries[rb.writeIdx] = entry
	rb.writeIdx = (rb.writeIdx + 1) % len(rb.entries)
	rb.memUsed += int64(entry.size) - evictedSize
	return evictedSize
}

// read returns entries in chronological order.
func (rb *ringBuffer) read() []LogEntry {
	if rb.count == 0 {
		return nil
	}
	result := make([]LogEntry, rb.count)
	if rb.count < len(rb.entries) {
		copy(result, rb.entries[:rb.count])
		return result
	}
	// Full buffer: oldest is at writeIdx.
	n := copy(result, rb.entries[rb.writeIdx:])
	copy(result[n:], rb.entries[:rb.writeIdx])
	return result
}

// evictOldest removes the oldest entry and returns its size. Returns 0 if empty.
func (rb *ringBuffer) evictOldest() int64 {
	if rb.count == 0 {
		return 0
	}
	oldestIdx := rb.writeIdx - rb.count
	if oldestIdx < 0 {
		oldestIdx += len(rb.entries)
	}
	evicted := rb.entries[oldestIdx]
	rb.entries[oldestIdx] = LogEntry{} // release references
	rb.count--
	rb.memUsed -= int64(evicted.size)
	return int64(evicted.size)
}

const defaultRingSize = 1024

// LogStore holds per-probe ring buffers with a total memory ceiling.
type LogStore struct {
	mu            sync.RWMutex
	probeBuffers  map[string]*ringBuffer
	globalBuffer  *ringBuffer
	maxMemBytes   int64
	curMemBytes   int64
	minStoreLevel slog.Level
}

// New creates a new LogStore with the given memory ceiling and minimum log level.
func New(maxMemBytes int64, minLevel slog.Level) *LogStore {
	return &LogStore{
		probeBuffers:  make(map[string]*ringBuffer),
		globalBuffer:  &ringBuffer{entries: make([]LogEntry, defaultRingSize)},
		maxMemBytes:   maxMemBytes,
		minStoreLevel: minLevel,
	}
}

// sourceFromAttrs returns the source identifier for a log entry.
// It prefers "probe", then falls back to "component", then "system".
func sourceFromAttrs(attrs map[string]string) string {
	if v := attrs["probe"]; v != "" {
		return v
	}
	if v := attrs["component"]; v != "" {
		return v
	}
	if v := attrs["system"]; v != "" {
		return v
	}
	return ""
}

// Store adds a log entry to the store. It extracts the source (probe name,
// component, or system) from attrs to determine which buffer to use.
func (ls *LogStore) Store(level slog.Level, msg string, attrs []slog.Attr) {
	if level < ls.minStoreLevel {
		return
	}

	flatAttrs := make(map[string]string, len(attrs))
	for _, a := range attrs {
		flatAttrs[a.Key] = a.Value.String()
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Attrs:     flatAttrs,
	}
	entry.size = estimateSize(msg, flatAttrs)

	source := sourceFromAttrs(flatAttrs)

	ls.mu.Lock()
	defer ls.mu.Unlock()

	rb := ls.getOrCreateBuffer(source)
	evictedSize := rb.write(entry)

	ls.curMemBytes += int64(entry.size)
	if evictedSize > 0 {
		// The ring buffer itself evicted an entry (buffer was full).
		ls.curMemBytes -= evictedSize
	}

	// Only evict when over the ceiling. When triggered, evict down to 95%
	// to create headroom and amortize the cost of scanning all buffers.
	if ls.curMemBytes > ls.maxMemBytes {
		ls.evictTo(ls.maxMemBytes * 95 / 100)
	}
}

func (ls *LogStore) getOrCreateBuffer(source string) *ringBuffer {
	if source == "" {
		return ls.globalBuffer
	}
	rb, ok := ls.probeBuffers[source]
	if !ok {
		rb = &ringBuffer{entries: make([]LogEntry, defaultRingSize)}
		ls.probeBuffers[source] = rb
	}
	return rb
}

// evictTo evicts oldest entries from the buffer with the most memory usage
// until total memory is at or below the target. Must be called with ls.mu held.
func (ls *LogStore) evictTo(target int64) {
	for ls.curMemBytes > target {
		// Find the buffer using the most memory.
		var maxBuf *ringBuffer
		var maxMem int64

		if ls.globalBuffer.memUsed > maxMem && ls.globalBuffer.count > 0 {
			maxBuf = ls.globalBuffer
			maxMem = ls.globalBuffer.memUsed
		}
		for _, rb := range ls.probeBuffers {
			if rb.memUsed > maxMem && rb.count > 0 {
				maxBuf = rb
				maxMem = rb.memUsed
			}
		}
		if maxBuf == nil {
			break // nothing to evict
		}
		freed := maxBuf.evictOldest()
		ls.curMemBytes -= freed
	}
}

// QueryOpts specifies filters for querying log entries.
type QueryOpts struct {
	Source   string // probe name, component name, or empty for all
	MinLevel slog.Level
	Since    time.Time
	Until    time.Time
	Limit    int
}

// Query returns log entries matching the given filters, in chronological order.
func (ls *LogStore) Query(opts QueryOpts) []LogEntry {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	var allEntries []LogEntry

	if opts.Source != "" {
		rb, ok := ls.probeBuffers[opts.Source]
		if ok {
			allEntries = rb.read()
		}
	} else {
		// Collect from all buffers.
		for _, rb := range ls.probeBuffers {
			allEntries = append(allEntries, rb.read()...)
		}
		allEntries = append(allEntries, ls.globalBuffer.read()...)
	}

	// Filter.
	var filtered []LogEntry
	for _, e := range allEntries {
		if e.Level < opts.MinLevel {
			continue
		}
		if !opts.Since.IsZero() && e.Timestamp.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && e.Timestamp.After(opts.Until) {
			continue
		}
		filtered = append(filtered, e)
	}

	// Sort by timestamp (entries from different buffers may interleave).
	if opts.Source == "" && len(filtered) > 1 {
		sortByTimestamp(filtered)
	}

	if opts.Limit > 0 && len(filtered) > opts.Limit {
		// Return the most recent entries.
		filtered = filtered[len(filtered)-opts.Limit:]
	}

	return filtered
}

// SourceNames returns the names of all sources (probes, components) that
// have log entries.
func (ls *LogStore) SourceNames() []string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	names := make([]string, 0, len(ls.probeBuffers))
	for name := range ls.probeBuffers {
		names = append(names, name)
	}
	return names
}

func sortByTimestamp(entries []LogEntry) {
	slices.SortFunc(entries, func(a, b LogEntry) int {
		return a.Timestamp.Compare(b.Timestamp)
	})
}
