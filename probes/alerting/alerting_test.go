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
	"strconv"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertHandlerRecord(t *testing.T) {
	alertInfo := func(target string, failures, total, dur int) *AlertInfo {
		return &AlertInfo{
			Name:         "test-probe",
			ProbeName:    "test-probe",
			Target:       endpoint.Endpoint{Name: target},
			Failures:     failures,
			Total:        total,
			FailingSince: time.Time{}.Add(time.Duration(dur) * time.Second),
			ConditionID:  strconv.FormatInt(time.Time{}.Add(time.Duration(dur)*time.Second).Unix(), 10),
		}
	}
	tests := []struct {
		name           string
		condition      *configpb.Condition
		targets        []string
		total, success [][]int64
		wantAlerted    []bool // wantAlerted[i] is true if target i should be alerted.
		wantAlerts     []*AlertInfo
		wantErr        bool
	}{
		{
			name:        "single-target-no-alert",
			targets:     []string{"target1"},
			total:       [][]int64{{1, 2}},
			success:     [][]int64{{1, 2}},
			wantAlerted: []bool{false},
		},
		{
			name:        "single-target-alert-default-condition",
			targets:     []string{"target1"},
			total:       [][]int64{{1, 2, 3}},
			success:     [][]int64{{1, 2, 2}}, // Success didn't increase.
			wantAlerted: []bool{true},
			wantAlerts:  []*AlertInfo{alertInfo("target1", 1, 1, 2)},
		},
		{
			name:        "default-condition-one-point-no-alert",
			targets:     []string{"target1"},
			total:       [][]int64{{2}},
			success:     [][]int64{{1}},
			wantAlerted: []bool{false},
		},
		{
			name:        "alerts-last-alert-cleared",
			targets:     []string{"target1"},
			total:       [][]int64{{2, 4, 6, 8}},
			success:     [][]int64{{1, 3, 4, 6}},
			wantAlerted: []bool{false},
			wantAlerts:  []*AlertInfo{alertInfo("target1", 1, 1, 2)},
		},
		{
			name:        "alert-over-a-period-of-time",
			condition:   &configpb.Condition{Failures: int32(3), Total: int32(5)},
			targets:     []string{"target1"},
			total:       [][]int64{{2, 4, 6, 8}}, // total: 2, 2, 2
			success:     [][]int64{{1, 2, 4, 4}}, // failures: 1, 0, 2
			wantAlerted: []bool{true},
			wantAlerts:  []*AlertInfo{alertInfo("target1", 3, 5, 3)},
		},
		{
			name:        "over-a-period-of-time-alert-cleared",
			condition:   &configpb.Condition{Failures: int32(3), Total: int32(5)},
			targets:     []string{"target1"},
			total:       [][]int64{{2, 4, 6, 8, 10}}, // total: 2, 2, 2, 2
			success:     [][]int64{{1, 2, 4, 4, 6}},  // failures: 1, 0, 2, 0
			wantAlerted: []bool{false},
			wantAlerts:  []*AlertInfo{alertInfo("target1", 3, 5, 3)},
		},
		{
			name:      "over-a-period-of-time-alert-cleared-alerted-again",
			condition: &configpb.Condition{Failures: int32(3), Total: int32(5)},
			targets:   []string{"target1"},
			total:     [][]int64{{2, 4, 6, 8, 10, 12}}, // total:    2, 2, 2, 2, 2
			success:   [][]int64{{1, 2, 4, 4, 6, 6}},   // failures: 1, 0, 2, 0, 2
			// total:    2, 2, 2, 2, 2
			// failures: 1, 0, 2, 0, 2
			wantAlerted: []bool{true},
			wantAlerts:  []*AlertInfo{alertInfo("target1", 3, 5, 3), alertInfo("target1", 3, 5, 5)},
		},
		{
			name:      "only-target2-alert",
			condition: &configpb.Condition{Failures: int32(2), Total: int32(0)},
			targets:   []string{"target1", "target2"},
			total:     [][]int64{{1, 2, 3, 4}, {1, 2, 3, 4}},
			success:   [][]int64{{1, 2, 2, 3}, {1, 2, 2, 2}},
			// total:    1, 1, 1 (target1); 1, 1, 1 (target2)
			// failures: 0, 1, 0 (target1); 0, 1, 1 (target2)
			wantAlerted: []bool{false, true},
			wantAlerts:  []*AlertInfo{alertInfo("target2", 2, 2, 3)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah := NewAlertHandler(&configpb.AlertConf{Condition: tt.condition}, "test-probe", nil)
			ah.notifyCh = make(chan *AlertInfo, 10)

			for i, target := range tt.targets {
				ep := endpoint.Endpoint{Name: target}
				ts := time.Time{}

				for j := range tt.total[i] {
					em := metrics.NewEventMetrics(ts)
					em.AddMetric("total", metrics.NewInt(tt.total[i][j]))
					em.AddMetric("success", metrics.NewInt(tt.success[i][j]))

					if err := ah.Record(ep, em); (err != nil) != tt.wantErr {
						t.Errorf("AlertHandler.Record() error = %v, wantErr %v", err, tt.wantErr)
					}
					t.Logf("target (%s) state: %+v", target, ah.targets[ep.Key()])

					ts = ts.Add(time.Second)
				}

				// Verify that target is in expected alerted state after the
				// run.
				assert.Equal(t, tt.wantAlerted[i], ah.targets[ep.Key()].alerted, target+" alerted")
			}

			// Verify that alerts are sent on the notify channel.
			assert.Equal(t, len(tt.wantAlerts), len(ah.notifyCh), "number of alerts")
			if len(tt.wantAlerts) == len(ah.notifyCh) {
				for i := range tt.wantAlerts {
					a := <-ah.notifyCh
					assert.Equal(t, tt.wantAlerts[i], a)
				}
			}
		})
	}
}

func TestNewAlertHandler(t *testing.T) {
	tests := []struct {
		name      string
		conf      *configpb.AlertConf
		probeName string
		want      *AlertHandler
	}{
		{
			name:      "default-condition",
			probeName: "test-probe",
			conf: &configpb.AlertConf{
				Name: "test-alert",
			},
			want: &AlertHandler{
				name:      "test-alert",
				probeName: "test-probe",
				condition: &configpb.Condition{Failures: 1, Total: 1},
				targets:   make(map[string]*targetState),
			},
		},
		{
			name:      "no-alert-name",
			probeName: "test-probe",
			conf: &configpb.AlertConf{
				Condition: &configpb.Condition{Failures: 4, Total: 5},
			},
			want: &AlertHandler{
				name:      "test-probe",
				probeName: "test-probe",
				condition: &configpb.Condition{Failures: 4, Total: 5},
				targets:   make(map[string]*targetState),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewAlertHandler(tt.conf, tt.probeName, nil))
		})
	}
}
