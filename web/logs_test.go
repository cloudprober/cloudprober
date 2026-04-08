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

package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/logger/logstore"
	"github.com/stretchr/testify/assert"
)

func setupLogStore(t *testing.T) *logstore.LogStore {
	t.Helper()
	ls := logstore.New(1<<20, slog.LevelInfo)

	old := logger.DefaultLogStore()
	logger.SetDefaultLogStore(ls)
	t.Cleanup(func() { logger.SetDefaultLogStore(old) })

	ls.Store(slog.LevelInfo, "info msg", []slog.Attr{slog.String("probe", "p1"), slog.String("target", "t1")})
	ls.Store(slog.LevelWarn, "warn msg", []slog.Attr{slog.String("probe", "p1")})
	ls.Store(slog.LevelError, "error msg", []slog.Attr{slog.String("probe", "p2")})

	return ls
}

func TestLogsHandlerNoStore(t *testing.T) {
	old := logger.DefaultLogStore()
	logger.SetDefaultLogStore(nil)
	defer logger.SetDefaultLogStore(old)

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestLogsHandlerHTML(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "info msg")
	assert.Contains(t, body, "warn msg")
	assert.Contains(t, body, "error msg")
	assert.Contains(t, body, "Logs")
}

func TestLogsHandlerHTMLFilterSource(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?source=p1", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "info msg")
	assert.Contains(t, body, "warn msg")
	assert.NotContains(t, body, "error msg")
}

func TestLogsHandlerHTMLFilterLevel(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?level=ERROR", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	assert.NotContains(t, body, "info msg")
	assert.NotContains(t, body, "warn msg")
	assert.Contains(t, body, "error msg")
}

func TestLogsHandlerHTMLLimit(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?limit=1", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	// Only the most recent entry should be present.
	assert.Contains(t, body, "error msg")
	assert.NotContains(t, body, "info msg")
}

func TestLogsHandlerHTMLEmpty(t *testing.T) {
	ls := logstore.New(1<<20, slog.LevelInfo)
	old := logger.DefaultLogStore()
	logger.SetDefaultLogStore(ls)
	defer logger.SetDefaultLogStore(old)

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "No log entries found")
}

func TestLogsHandlerJSON(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?format=json", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	respBody, err := io.ReadAll(w.Body)
	assert.NoError(t, err)

	var entries []logEntryJSON
	assert.NoError(t, json.Unmarshal(respBody, &entries))
	assert.Len(t, entries, 3)

	assert.Equal(t, "info msg", entries[0].Message)
	assert.Equal(t, "INFO", entries[0].Level)
	assert.Equal(t, "p1", entries[0].Attrs["probe"])

	assert.Equal(t, "error msg", entries[2].Message)
	assert.Equal(t, "ERROR", entries[2].Level)
}

func TestLogsHandlerJSONWithFilters(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?format=json&source=p2&level=ERROR", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	var entries []logEntryJSON
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Len(t, entries, 1)
	assert.Equal(t, "error msg", entries[0].Message)
}

func TestLogsHandlerHTMLAttrs(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs?source=p1", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	// target=t1 should be in the attrs column (not filtered as source key).
	assert.Contains(t, body, "target=t1")
}

func TestLogsHandlerSourceDropdown(t *testing.T) {
	setupLogStore(t)

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()
	logsHandler(w, req)

	body := w.Body.String()
	assert.Contains(t, body, `<option value="p1"`)
	assert.Contains(t, body, `<option value="p2"`)
}

func TestSourceLabel(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"probe", map[string]string{"probe": "p1"}, "p1"},
		{"component", map[string]string{"component": "rds"}, "rds"},
		{"system", map[string]string{"system": "cloudprober"}, "cloudprober"},
		{"probe precedence", map[string]string{"probe": "p1", "component": "c1"}, "p1"},
		{"empty", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sourceLabel(tt.attrs))
		})
	}
}
