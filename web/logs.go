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
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/logger/logstore"
	"github.com/cloudprober/cloudprober/web/resources"
)

var logsTmpl = template.Must(template.New("logs").Parse(`
<style>
  .logs-filters {
    background: #E1F6FF;
    padding: 8px 12px;
    border-radius: 4px;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .logs-filters label {
    font-weight: bold;
    font-size: 13px;
  }
  .logs-filters select {
    padding: 4px 8px;
    border: 1px solid #ccc;
    border-radius: 3px;
    font-size: 13px;
    background: white;
  }
  .logs-table {
    border-collapse: collapse;
    border-spacing: 0;
    width: 100%;
    font-family: monospace;
    font-size: 13px;
  }
  .logs-table th {
    background: #f0f0f0;
    text-align: left;
    padding: 6px 10px;
    border: 1px solid #ddd;
    white-space: nowrap;
  }
  .logs-table td {
    padding: 4px 10px;
    border: 1px solid #eee;
    max-width: 600px;
    word-wrap: break-word;
    vertical-align: top;
  }
  .logs-table tr:hover { background: #f8f8f8; }
  .logs-table .time { white-space: nowrap; color: #666; }
  .logs-table .level-ERROR, .logs-table .level-WARN { color: #c00; font-weight: bold; }
  .logs-table .level-DEBUG { color: #999; }
  .logs-table .source { color: #0066cc; }
  .logs-table .attrs { }
  .logs-empty { color: #999; margin-top: 20px; }
</style>

<h3>Logs</h3>

<form id="logsForm" method="get" action="/logs">
  <div class="logs-filters">
    <label>Source:
      <select name="source" onchange="this.form.submit()">
        <option value="">All</option>
        {{range .Sources}}
        <option value="{{.}}" {{if eq . $.Source}}selected{{end}}>{{.}}</option>
        {{end}}
      </select>
    </label>
    <label>Level:
      <select name="level" onchange="this.form.submit()">
        <option value="DEBUG" {{if eq .Level "DEBUG"}}selected{{end}}>DEBUG</option>
        <option value="INFO" {{if eq .Level "INFO"}}selected{{end}}>INFO</option>
        <option value="WARNING" {{if eq .Level "WARNING"}}selected{{end}}>WARNING</option>
        <option value="ERROR" {{if eq .Level "ERROR"}}selected{{end}}>ERROR</option>
      </select>
    </label>
    <label>Limit:
      <select name="limit" onchange="this.form.submit()">
        {{range .LimitOptions}}
        <option value="{{.}}" {{if eq . $.Limit}}selected{{end}}>{{.}}</option>
        {{end}}
      </select>
    </label>
  </div>
</form>

{{if .Entries}}
<table class="logs-table">
<thead>
<tr>
  <th>Time</th>
  <th>Level</th>
  <th>Source</th>
  <th>Message</th>
  <th>Attributes</th>
</tr>
</thead>
<tbody>
{{range .Entries}}
<tr>
  <td class="time">{{.Time}}</td>
  <td class="level-{{.Level}}">{{.Level}}</td>
  <td class="source">{{.Source}}</td>
  <td>{{.Message}}</td>
  <td class="attrs">{{.Attrs}}</td>
</tr>
{{end}}
</tbody>
</table>
{{else}}
<p class="logs-empty">No log entries found.</p>
{{end}}
`))

type logsPageData struct {
	Source       string
	Sources      []string
	Level        string
	Limit        int
	LimitOptions []int
	Entries      []logEntryView
}

type logEntryView struct {
	Time    string
	Level   string
	Source  string
	Message string
	Attrs   string
}

type logEntryJSON struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Attrs     map[string]string `json:"attrs"`
}

// sourceLabel returns the display label for a log entry's source.
func sourceLabel(attrs map[string]string) string {
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

func logsHandler(w http.ResponseWriter, r *http.Request) {
	ls := logger.DefaultLogStore()
	if ls == nil {
		http.Error(w, "log store is not enabled", http.StatusServiceUnavailable)
		return
	}

	source := r.URL.Query().Get("source")
	levelStr := r.URL.Query().Get("level")
	if levelStr == "" {
		levelStr = "INFO"
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = time.Unix(v, 0)
		}
	}

	entries := ls.Query(logstore.QueryOpts{
		Source:   source,
		MinLevel: logger.ParseLogLevel(levelStr),
		Since:    since,
		Limit:    limit,
	})

	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		jsonEntries := make([]logEntryJSON, len(entries))
		for i, e := range entries {
			jsonEntries[i] = logEntryJSON{
				Timestamp: e.Timestamp.Format(time.RFC3339Nano),
				Level:     e.Level.String(),
				Message:   e.Message,
				Attrs:     e.Attrs,
			}
		}
		json.NewEncoder(w).Encode(jsonEntries)
		return
	}

	// Build source dropdown options.
	sourceNames := ls.SourceNames()
	sort.Strings(sourceNames)

	views := make([]logEntryView, len(entries))
	for i, e := range entries {
		// Build a compact attrs string excluding source-related keys.
		var attrParts []string
		for k, v := range e.Attrs {
			if k == "probe" || k == "system" || k == "component" {
				continue
			}
			attrParts = append(attrParts, k+"="+v)
		}
		sort.Strings(attrParts)

		views[i] = logEntryView{
			Time:    e.Timestamp.Format("15:04:05.000"),
			Level:   e.Level.String(),
			Source:  sourceLabel(e.Attrs),
			Message: e.Message,
			Attrs:   strings.Join(attrParts, ", "),
		}
	}

	data := logsPageData{
		Source:       source,
		Sources:      sourceNames,
		Level:        strings.ToUpper(levelStr),
		Limit:        limit,
		LimitOptions: []int{50, 100, 200, 500, 1000},
		Entries:      views,
	}

	body := resources.ExecTmpl(logsTmpl, data)
	w.Write([]byte(resources.RenderPage("/logs", body)))
}
