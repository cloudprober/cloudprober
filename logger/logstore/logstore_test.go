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

package logstore

import (
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestStoreAndQuery(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo) // 1MB ceiling

	ls.Store(slog.LevelInfo, "probe1 msg1", []slog.Attr{slog.String("probe", "p1")})
	ls.Store(slog.LevelInfo, "probe2 msg1", []slog.Attr{slog.String("probe", "p2")})
	ls.Store(slog.LevelError, "probe1 err", []slog.Attr{slog.String("probe", "p1")})

	// Query all.
	entries := ls.Query(QueryOpts{})
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Query by probe.
	entries = ls.Query(QueryOpts{Source: "p1"})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for p1, got %d", len(entries))
	}

	// Query by level.
	entries = ls.Query(QueryOpts{MinLevel: slog.LevelError})
	if len(entries) != 1 {
		t.Fatalf("expected 1 error entry, got %d", len(entries))
	}
	if entries[0].Message != "probe1 err" {
		t.Errorf("expected 'probe1 err', got %q", entries[0].Message)
	}
}

func TestMinStoreLevel(t *testing.T) {
	ls := New(1<<20, slog.LevelWarn)

	ls.Store(slog.LevelInfo, "info msg", []slog.Attr{slog.String("probe", "p1")})
	ls.Store(slog.LevelWarn, "warn msg", []slog.Attr{slog.String("probe", "p1")})

	entries := ls.Query(QueryOpts{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (info filtered at store level), got %d", len(entries))
	}
}

func TestRingBufferEviction(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	// Write more than defaultRingSize entries to one probe.
	for i := 0; i < defaultRingSize+100; i++ {
		ls.Store(slog.LevelInfo, fmt.Sprintf("msg %d", i), []slog.Attr{slog.String("probe", "p1")})
	}

	entries := ls.Query(QueryOpts{Source: "p1"})
	if len(entries) != defaultRingSize {
		t.Fatalf("expected %d entries (ring buffer size), got %d", defaultRingSize, len(entries))
	}

	// Oldest entries should have been evicted — first entry should be msg 100.
	if entries[0].Message != "msg 100" {
		t.Errorf("expected oldest entry 'msg 100', got %q", entries[0].Message)
	}
}

func TestMemoryCeiling(t *testing.T) {
	// Set a very small memory ceiling.
	ls := New(2000, slog.LevelInfo)

	for i := 0; i < 100; i++ {
		ls.Store(slog.LevelInfo, fmt.Sprintf("msg %d", i), []slog.Attr{slog.String("probe", "p1")})
	}

	ls.mu.RLock()
	mem := ls.curMemBytes
	ls.mu.RUnlock()

	// Allow up to 5% overshoot (eviction triggers at 105%, targets 95%).
	if mem > 2100 {
		t.Errorf("memory usage %d exceeds ceiling 2000 (with 5%% overshoot)", mem)
	}

	entries := ls.Query(QueryOpts{Source: "p1"})
	if len(entries) == 0 {
		t.Error("expected at least some entries")
	}
	if len(entries) >= 100 {
		t.Error("expected some entries to have been evicted due to memory ceiling")
	}
}

func TestQueryLimit(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	for i := 0; i < 50; i++ {
		ls.Store(slog.LevelInfo, fmt.Sprintf("msg %d", i), []slog.Attr{slog.String("probe", "p1")})
	}

	entries := ls.Query(QueryOpts{Limit: 10})
	if len(entries) != 10 {
		t.Fatalf("expected 10 entries with limit, got %d", len(entries))
	}
	// Should return the most recent 10.
	if entries[0].Message != "msg 40" {
		t.Errorf("expected 'msg 40', got %q", entries[0].Message)
	}
}

func TestQueryTimeRange(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	now := time.Now()

	ls.Store(slog.LevelInfo, "old", []slog.Attr{slog.String("probe", "p1")})
	// Entries get time.Now() timestamp, so we filter using Since.
	entries := ls.Query(QueryOpts{Since: now.Add(-time.Second)})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entries = ls.Query(QueryOpts{Since: now.Add(time.Hour)})
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for future Since, got %d", len(entries))
	}
}

func TestGlobalBuffer(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	// Store without probe attr.
	ls.Store(slog.LevelInfo, "global msg", []slog.Attr{slog.String("component", "main")})

	entries := ls.Query(QueryOpts{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 global entry, got %d", len(entries))
	}
	if entries[0].Attrs["component"] != "main" {
		t.Errorf("expected component=main, got %v", entries[0].Attrs)
	}
}

func TestSourceNames(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	ls.Store(slog.LevelInfo, "a", []slog.Attr{slog.String("probe", "p1")})
	ls.Store(slog.LevelInfo, "b", []slog.Attr{slog.String("probe", "p2")})

	names := ls.SourceNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 probe names, got %d", len(names))
	}
}

func TestSourceFromAttrs(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"probe", map[string]string{"probe": "p1"}, "p1"},
		{"component", map[string]string{"component": "rds-server"}, "rds-server"},
		{"system", map[string]string{"system": "cloudprober"}, "cloudprober"},
		{"probe takes precedence", map[string]string{"probe": "p1", "component": "c1", "system": "s1"}, "p1"},
		{"component over system", map[string]string{"component": "c1", "system": "s1"}, "c1"},
		{"empty", map[string]string{}, ""},
		{"unrelated keys", map[string]string{"foo": "bar"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourceFromAttrs(tt.attrs); got != tt.want {
				t.Errorf("sourceFromAttrs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryUntil(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	ls.Store(slog.LevelInfo, "msg", []slog.Attr{slog.String("probe", "p1")})

	// Until in the past should filter out the entry.
	entries := ls.Query(QueryOpts{Until: time.Now().Add(-time.Hour)})
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for past Until, got %d", len(entries))
	}

	// Until in the future should include the entry.
	entries = ls.Query(QueryOpts{Until: time.Now().Add(time.Hour)})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for future Until, got %d", len(entries))
	}
}

func TestQueryNonexistentSource(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	ls.Store(slog.LevelInfo, "msg", []slog.Attr{slog.String("probe", "p1")})

	entries := ls.Query(QueryOpts{Source: "nonexistent"})
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for nonexistent source, got %d", len(entries))
	}
}

func TestGlobalBufferNoSourceAttrs(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	// Entry with no probe/component/system goes to global buffer.
	ls.Store(slog.LevelInfo, "global msg", []slog.Attr{slog.String("foo", "bar")})

	entries := ls.Query(QueryOpts{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Message != "global msg" {
		t.Errorf("expected 'global msg', got %q", entries[0].Message)
	}
}

func TestMemoryCeilingMultipleProbes(t *testing.T) {
	// Test that eviction works across multiple probe buffers and the global buffer.
	ls := New(3000, slog.LevelInfo)

	// Store to multiple sources including global.
	for i := 0; i < 50; i++ {
		ls.Store(slog.LevelInfo, fmt.Sprintf("p1 msg %d", i), []slog.Attr{slog.String("probe", "p1")})
		ls.Store(slog.LevelInfo, fmt.Sprintf("p2 msg %d", i), []slog.Attr{slog.String("probe", "p2")})
		ls.Store(slog.LevelInfo, fmt.Sprintf("global msg %d", i), []slog.Attr{slog.String("foo", "bar")})
	}

	ls.mu.RLock()
	mem := ls.curMemBytes
	ls.mu.RUnlock()

	if mem > 3150 { // 5% overshoot
		t.Errorf("memory usage %d exceeds ceiling 3000 (with 5%% overshoot)", mem)
	}

	// All sources should have some entries.
	p1 := ls.Query(QueryOpts{Source: "p1"})
	p2 := ls.Query(QueryOpts{Source: "p2"})
	all := ls.Query(QueryOpts{})

	if len(p1) == 0 || len(p2) == 0 {
		t.Errorf("expected entries for both probes, got p1=%d p2=%d", len(p1), len(p2))
	}
	if len(all) == 0 {
		t.Error("expected some entries in total")
	}
}

func TestEvictOldestEmpty(t *testing.T) {
	rb := &ringBuffer{entries: make([]LogEntry, 10)}
	if got := rb.evictOldest(); got != 0 {
		t.Errorf("evictOldest on empty buffer returned %d, want 0", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	ls := New(1<<20, slog.LevelInfo)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(probe string) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				ls.Store(slog.LevelInfo, fmt.Sprintf("msg %d", j), []slog.Attr{slog.String("probe", probe)})
			}
		}(fmt.Sprintf("p%d", i))
	}

	// Concurrent reads.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ls.Query(QueryOpts{})
			}
		}()
	}

	wg.Wait()

	entries := ls.Query(QueryOpts{})
	if len(entries) == 0 {
		t.Error("expected entries after concurrent writes")
	}
}
