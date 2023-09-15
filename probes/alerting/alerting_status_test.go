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

	"github.com/cloudprober/cloudprober/probes/alerting/notifier"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestUpdateGlobalState(t *testing.T) {
	resetGlobalState()

	ah := [2]*AlertHandler{
		{
			name:      "test-alert-1",
			probeName: "test-probe-2",
		},
		{
			name:      "test-alert-2",
			probeName: "test-probe-2",
		},
	}

	alertInfo := func(handlerIndex int, target string, delay int) *notifier.AlertInfo {
		return &notifier.AlertInfo{
			Name:         ah[handlerIndex].name,
			ProbeName:    ah[handlerIndex].probeName,
			Target:       endpoint.Endpoint{Name: target},
			FailingSince: time.Time{}.Add(time.Duration(delay) * time.Second),
		}
	}

	tests := []struct {
		name           string
		addAlerts      [2][]*notifier.AlertInfo
		deleteAlerts   [2][]endpoint.Endpoint
		wantCurrAlerts []*notifier.AlertInfo
		wantPrevAlerts []*notifier.AlertInfo
	}{
		{
			name: "add-alerts",
			addAlerts: [2][]*notifier.AlertInfo{
				{
					alertInfo(0, "target1", 1),
					alertInfo(0, "target2", 3),
				},
				{
					alertInfo(1, "target1", 2),
				},
			},
			wantCurrAlerts: []*notifier.AlertInfo{
				alertInfo(0, "target1", 1),
				alertInfo(1, "target1", 2),
				alertInfo(0, "target2", 3),
			},
		},
		{
			name: "delete-alerts",
			deleteAlerts: [2][]endpoint.Endpoint{
				{},
				{
					endpoint.Endpoint{Name: "target1"},
				},
			},
			wantCurrAlerts: []*notifier.AlertInfo{
				alertInfo(0, "target1", 1),
				alertInfo(0, "target2", 3),
			},
			wantPrevAlerts: []*notifier.AlertInfo{
				alertInfo(1, "target1", 2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, ah := range ah {
				for _, ai := range tt.addAlerts[i] {
					updateGlobalState(ah.globalKey(ai.Target), ai)
				}
				for _, ep := range tt.deleteAlerts[i] {
					updateGlobalState(ah.globalKey(ep), nil)
				}
			}
			curr, prev := currentState()
			assert.Equal(t, tt.wantCurrAlerts, curr, "current alerts")

			var prevAlerts []*notifier.AlertInfo
			for _, ra := range prev {
				prevAlerts = append(prevAlerts, ra.AlertInfo)
			}
			assert.Equal(t, tt.wantPrevAlerts, prevAlerts, "previous alerts")
		})
	}
}

func TestStatusHTML(t *testing.T) {
	resetGlobalState()

	ah := &AlertHandler{
		name:      "test-alert-1",
		probeName: "test-probe-2",
	}

	alertInfo := func(target string, delay int) *notifier.AlertInfo {
		return &notifier.AlertInfo{
			Name:         ah.name,
			ProbeName:    ah.probeName,
			Target:       endpoint.Endpoint{Name: target},
			FailingSince: time.Time{}.Add(time.Duration(delay) * time.Second),
		}
	}

	tests := []struct {
		name           string
		addAlerts      []*notifier.AlertInfo
		deleteAlerts   []endpoint.Endpoint
		wantStatusHTML string
	}{
		{
			name: "add-alerts",
			addAlerts: []*notifier.AlertInfo{
				alertInfo("target1", 1),
				alertInfo("target2", 3),
			},
			wantStatusHTML: `
<h3>Current Alerts:</h3>

<table class="status-list">
<tr>
  <th>Failing Since</th>
  <th>Probe</th>
  <th>Target</th>
  <th>Condition ID</th>
  <th>Failures / Total</th>
</tr>

<tr>
  <td>0001-01-01 00:00:01 &#43;0000 UTC</td>
  <td>test-probe-2</td>
  <td>target1</td>
  <td></td>
  <td>0 / 0</td>
</tr>
<tr>
  <td>0001-01-01 00:00:03 &#43;0000 UTC</td>
  <td>test-probe-2</td>
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
			deleteAlerts: []endpoint.Endpoint{endpoint.Endpoint{Name: "target1"}},
			wantStatusHTML: `
<h3>Current Alerts:</h3>

<table class="status-list">
<tr>
  <th>Failing Since</th>
  <th>Probe</th>
  <th>Target</th>
  <th>Condition ID</th>
  <th>Failures / Total</th>
</tr>

<tr>
  <td>0001-01-01 00:00:03 &#43;0000 UTC</td>
  <td>test-probe-2</td>
  <td>target2</td>
  <td></td>
  <td>0 / 0</td>
</tr>
</table>

<h3>Alerts History:</h3>

<table class="status-list">
<tr>
  <th>Started At</th>
  <th>Probe</th>
  <th>Target</th>
  <th>Condition ID</th>
  <th>Resolved At</th>
</tr>

<tr>
  <td>0001-01-01 00:00:01 &#43;0000 UTC</td>
  <td>test-probe-2</td>
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
				updateGlobalState(ah.globalKey(ai.Target), ai)
			}
			for _, ep := range tt.deleteAlerts {
				updateGlobalState(ah.globalKey(ep), nil)
			}

			global.mu.Lock()
			for i := range global.previousAlerts {
				global.previousAlerts[i].ResolvedAt = time.Time{}.Add(4 * time.Second)
			}
			global.mu.Unlock()

			status, err := StatusHTML()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatusHTML, status)
		})
	}
}
