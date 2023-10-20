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
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestUpdateState(t *testing.T) {
	st := state{}

	oldMaxAlertsHistory := maxAlertsHistory
	maxAlertsHistory = 2
	defer func() { maxAlertsHistory = oldMaxAlertsHistory }()

	ah := [2]*AlertHandler{
		{
			name:      "test-alert-1",
			probeName: "test-probe-1",
		},
		{
			name:      "test-alert-2",
			probeName: "test-probe-2",
		},
	}

	alert := func(handlerIndex int, target string, delay int) *alertinfo.AlertInfo {
		return &alertinfo.AlertInfo{
			Name:         ah[handlerIndex].name,
			ProbeName:    ah[handlerIndex].probeName,
			Target:       endpoint.Endpoint{Name: target},
			FailingSince: time.Time{}.Add(time.Duration(delay) * time.Second),
		}
	}

	tests := []struct {
		name           string
		addAlerts      [2][]*alertinfo.AlertInfo
		deleteAlerts   [2][]string
		wantCurrAlerts []*alertinfo.AlertInfo
		wantPrevAlerts []*alertinfo.AlertInfo
	}{
		{
			name: "add-alerts",
			addAlerts: [2][]*alertinfo.AlertInfo{
				{
					alert(0, "target1", 1),
					alert(0, "target2", 3),
				},
				{
					alert(1, "target1", 2),
				},
			},
			wantCurrAlerts: []*alertinfo.AlertInfo{
				alert(0, "target1", 1),
				alert(1, "target1", 2),
				alert(0, "target2", 3),
			},
		},
		{
			name: "delete-alerts-1",
			deleteAlerts: [2][]string{
				{"target2"},
				{"target1"},
			},
			wantCurrAlerts: []*alertinfo.AlertInfo{
				alert(0, "target1", 1),
			},
			wantPrevAlerts: []*alertinfo.AlertInfo{
				alert(1, "target1", 2),
				alert(0, "target2", 3),
			},
		},
		{
			name: "delete-alerts-2",
			deleteAlerts: [2][]string{
				{"target1"},
				{},
			},
			wantPrevAlerts: []*alertinfo.AlertInfo{
				alert(0, "target1", 1),
				alert(1, "target1", 2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, ah := range ah {
				for _, ai := range tt.addAlerts[i] {
					st.add(ah.globalKey(ai.Target), ai)
				}
				for _, tgt := range tt.deleteAlerts[i] {
					st.resolve(ah.globalKey(endpoint.Endpoint{Name: tgt}))
				}
			}
			curr, prev := st.list()
			assert.Equal(t, tt.wantCurrAlerts, curr, "current alerts")

			var prevAlerts []*alertinfo.AlertInfo
			for _, ra := range prev {
				prevAlerts = append(prevAlerts, ra.AlertInfo)
			}
			assert.Equal(t, tt.wantPrevAlerts, prevAlerts, "previous alerts")
		})
	}
}

func TestStatusHTML(t *testing.T) {
	oldGlobalState := globalState
	globalState = state{}
	defer func() { globalState = oldGlobalState }()

	ah := &AlertHandler{
		name:      "test-alert-1",
		probeName: "test-probe-2",
	}

	alertInfo := func(target string, delay int) *alertinfo.AlertInfo {
		return &alertinfo.AlertInfo{
			Name:         ah.name,
			ProbeName:    ah.probeName,
			Target:       endpoint.Endpoint{Name: target},
			FailingSince: time.Time{}.Add(time.Duration(delay) * time.Second),
		}
	}

	tests := []struct {
		name           string
		addAlerts      []*alertinfo.AlertInfo
		deleteAlerts   []endpoint.Endpoint
		wantStatusHTML string
	}{
		{
			name: "add-alerts",
			addAlerts: []*alertinfo.AlertInfo{
				alertInfo("target1", 1),
				alertInfo("target2", 3),
			},
			wantStatusHTML: `
<h3>Current Alerts:</h3>

<table class="status-list">
<tr>
  <th>Alert</th>
  <th>Probe</th>
  <th>Failing Since</th>
  <th>Target</th>
  <th>Deduplication ID</th>
  <th>Failures / Total</th>
</tr>

<tr>
  <td>test-alert-1</td>
  <td>test-probe-2</td>
  <td>0001-01-01 00:00:01 &#43;0000 UTC</td>
  <td>target1</td>
  <td></td>
  <td>0 / 0</td>
</tr>
<tr>
  <td>test-alert-1</td>
  <td>test-probe-2</td>
  <td>0001-01-01 00:00:03 &#43;0000 UTC</td>
  <td>target2</td>
  <td></td>
  <td>0 / 0</td>
</tr>
</table>

<h3>Alerts History:</h3>

  <p>No alerts.</p>

`,
		},
		{
			name:         "delete-alerts",
			deleteAlerts: []endpoint.Endpoint{{Name: "target1"}},
			wantStatusHTML: `
<h3>Current Alerts:</h3>

<table class="status-list">
<tr>
  <th>Alert</th>
  <th>Probe</th>
  <th>Failing Since</th>
  <th>Target</th>
  <th>Deduplication ID</th>
  <th>Failures / Total</th>
</tr>

<tr>
  <td>test-alert-1</td>
  <td>test-probe-2</td>
  <td>0001-01-01 00:00:03 &#43;0000 UTC</td>
  <td>target2</td>
  <td></td>
  <td>0 / 0</td>
</tr>
</table>

<h3>Alerts History:</h3>

<p>[Showing last 1 resolved alerts..]</p>
<table class="status-list">
<tr>
  <th>Alert</th>
  <th>Probe</th>
  <th>Started At</th>
  <th>Target</th>
  <th>Deduplication ID</th>
  <th>Resolved At</th>
</tr>

<tr>
  <td>test-alert-1</td>
  <td>test-probe-2</td>
  <td>0001-01-01 00:00:01 &#43;0000 UTC</td>
  <td>target1</td>
  <td></td>
  <td>0001-01-01 00:00:04 &#43;0000 UTC</td>
</tr>
</table>
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, ai := range tt.addAlerts {
				globalState.add(ah.globalKey(ai.Target), ai)
			}
			for _, ep := range tt.deleteAlerts {
				globalState.resolve(ah.globalKey(ep))
			}

			globalState.mu.Lock()
			for i := range globalState.resolvedAlerts {
				globalState.resolvedAlerts[i].ResolvedAt = time.Time{}.Add(4 * time.Second)
			}
			globalState.mu.Unlock()

			statusHTML, err := StatusHTML()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatusHTML, statusHTML)
		})
	}
}
