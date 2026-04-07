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
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/logger/logstore"
	"github.com/cloudprober/cloudprober/web/resources"
)

var logsTmpl = template.Must(template.New("logs").Parse(`
<h3>Probe Logs</h3>

<form method="get" action="/logs" style="margin-bottom: 1em;">
  <label>Probe: <input type="text" name="probe" value="{{.Probe}}" placeholder="all"></label>
  <label>Level:
    <select name="level">
      <option value="DEBUG" {{if eq .Level "DEBUG"}}selected{{end}}>DEBUG</option>
      <option value="INFO" {{if eq .Level "INFO"}}selected{{end}}>INFO</option>
      <option value="WARNING" {{if eq .Level "WARNING"}}selected{{end}}>WARNING</option>
      <option value="ERROR" {{if eq .Level "ERROR"}}selected{{end}}>ERROR</option>
    </select>
  </label>
  <label>Limit: <input type="number" name="limit" value="{{.Limit}}" style="width:60px"></label>
  <button type="submit">Filter</button>
</form>

{{if .Entries}}
<table border="1" cellpadding="4" cellspacing="0" style="border-collapse:collapse; font-size:0.9em;">
<tr>
  <th>Time</th>
  <th>Level</th>
  <th>Probe</th>
  <th>Message</th>
  <th>Attributes</th>
</tr>
{{range .Entries}}
<tr>
  <td style="white-space:nowrap">{{.Time}}</td>
  <td>{{.Level}}</td>
  <td>{{.Probe}}</td>
  <td>{{.Message}}</td>
  <td style="font-size:0.8em">{{.Attrs}}</td>
</tr>
{{end}}
</table>
{{else}}
<p>No log entries found.</p>
{{end}}
`))

type logsPageData struct {
	Probe   string
	Level   string
	Limit   int
	Entries []logEntryView
}

type logEntryView struct {
	Time    string
	Level   string
	Probe   string
	Message string
	Attrs   string
}

type logEntryJSON struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Attrs     map[string]string `json:"attrs"`
}


func logsHandler(w http.ResponseWriter, r *http.Request) {
	ls := logger.DefaultLogStore()
	if ls == nil {
		http.Error(w, "log store is not enabled", http.StatusServiceUnavailable)
		return
	}

	probe := r.URL.Query().Get("probe")
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
		ProbeName: probe,
		MinLevel:  logger.ParseLogLevel(levelStr),
		Since:     since,
		Limit:     limit,
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

	views := make([]logEntryView, len(entries))
	for i, e := range entries {
		probeName := e.Attrs["probe"]
		// Build a compact attrs string excluding probe.
		var attrParts []string
		for k, v := range e.Attrs {
			if k == "probe" || k == "system" {
				continue
			}
			attrParts = append(attrParts, k+"="+v)
		}

		views[i] = logEntryView{
			Time:    e.Timestamp.Format("15:04:05.000"),
			Level:   e.Level.String(),
			Probe:   probeName,
			Message: e.Message,
			Attrs:   strings.Join(attrParts, ", "),
		}
	}

	data := logsPageData{
		Probe:   probe,
		Level:   strings.ToUpper(levelStr),
		Limit:   limit,
		Entries: views,
	}

	body := resources.ExecTmpl(logsTmpl, data)
	w.Write([]byte(resources.RenderPage("/logs", body)))
}
