// Copyright 2023 The Cloudprober Authors.
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

package alerting

import (
	"bytes"
	"html/template"
	"sort"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
)

var statusTmpl = template.Must(template.New("status").Parse(`
<h3>Current Alerts:</h3>
{{ if not .CurrentAlerts }}
  <p>No alerts.</p>
{{ else }}
<table class="status-list">
<tr>
  <th>Alert</th>
  <th>Probe</th>
  <th>Failing Since</th>
  <th>Target</th>
  <th>Deduplication ID</th>
  <th>Failures / Total</th>
</tr>
{{ range .CurrentAlerts }}
<tr>
  <td>{{ .Name }}</td>
  <td>{{ .ProbeName }}</td>
  <td>{{ .FailingSince }}</td>
  <td>{{ .Target.Dst }}</td>
  <td>{{ .DeduplicationID }}</td>
  <td>{{ .Failures }} / {{ .Total }}</td>
</tr>
{{- end }}
</table>
{{- end }}

<h3>Alerts History:</h3>
{{ if not .PreviousAlerts }}
  <p>No alerts.</p>
{{ else }}
<p>[Showing last {{ len .PreviousAlerts }} resolved alerts..]</p>
<table class="status-list">
<tr>
  <th>Alert</th>
  <th>Probe</th>
  <th>Started At</th>
  <th>Target</th>
  <th>Deduplication ID</th>
  <th>Resolved At</th>
</tr>
{{ range .PreviousAlerts }}
<tr>
  <td>{{ .AlertInfo.Name }}</td>
  <td>{{ .AlertInfo.ProbeName }}</td>
  <td>{{ .AlertInfo.FailingSince }}</td>
  <td>{{ .AlertInfo.Target.Dst }}</td>
  <td>{{ .AlertInfo.DeduplicationID }}</td>
  <td>{{ .ResolvedAt }}</td>
</tr>
{{- end }}
</table>
{{- end }}
`))

// resolvedAlert is used to keep track of resolved alerts, to be able to show
// alerts hitory on the alerts dashboard.
type resolvedAlert struct {
	AlertInfo  *alertinfo.AlertInfo
	ResolvedAt time.Time
}

var maxAlertsHistory = 20

type state struct {
	mu             sync.RWMutex
	currentAlerts  map[string]*alertinfo.AlertInfo
	resolvedAlerts []resolvedAlert
}

func (st *state) get(key string) *alertinfo.AlertInfo {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.currentAlerts[key]
}

func (st *state) add(key string, ai *alertinfo.AlertInfo) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.currentAlerts == nil {
		st.currentAlerts = make(map[string]*alertinfo.AlertInfo)
	}
	st.currentAlerts[key] = ai
}

func (st *state) resolve(key string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	ra := resolvedAlert{st.currentAlerts[key], time.Now().Truncate(time.Second)}
	st.resolvedAlerts = append([]resolvedAlert{ra}, st.resolvedAlerts...)
	if len(st.resolvedAlerts) > maxAlertsHistory {
		st.resolvedAlerts = st.resolvedAlerts[:maxAlertsHistory]
	}
	delete(st.currentAlerts, key)
}

func (st *state) list() ([]*alertinfo.AlertInfo, []resolvedAlert) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var currentAlerts []*alertinfo.AlertInfo
	for _, ai := range st.currentAlerts {
		currentAlerts = append(currentAlerts, ai)
	}

	sort.Slice(currentAlerts, func(i, j int) bool {
		return currentAlerts[i].FailingSince.Before(currentAlerts[j].FailingSince)
	})

	return currentAlerts, append([]resolvedAlert{}, st.resolvedAlerts...)
}

func (st *state) statusHTML() (string, error) {
	var statusBuf bytes.Buffer

	currentAlerts, previousAlerts := st.list()
	for _, ai := range currentAlerts {
		ai.FailingSince = ai.FailingSince.Truncate(time.Second)
	}
	for _, a := range previousAlerts {
		a.AlertInfo.FailingSince = a.AlertInfo.FailingSince.Truncate(time.Second)
	}

	err := statusTmpl.Execute(&statusBuf, struct {
		CurrentAlerts  []*alertinfo.AlertInfo
		PreviousAlerts []resolvedAlert
	}{
		CurrentAlerts: currentAlerts, PreviousAlerts: previousAlerts,
	})

	return statusBuf.String(), err
}

var globalState = state{
	currentAlerts: make(map[string]*alertinfo.AlertInfo),
}

func StatusHTML() (string, error) {
	return globalState.statusHTML()
}
