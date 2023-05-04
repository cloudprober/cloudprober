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

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertHandlerRecord(t *testing.T) {
	tests := []struct {
		name              string
		failureThreshold  float32
		durationThreshold time.Duration
		targets           []string
		total, success    [][]int64
		wantAlerted       []bool // wantAlerted[i] is true if target i should be alerted.
		wantAlerts        []*AlertInfo
		wantErr           bool
	}{
		{
			name:             "single-target-no-alert",
			failureThreshold: 0.5,
			targets:          []string{"target1"},
			total:            [][]int64{{1, 2}},
			success:          [][]int64{{1, 2}},
			wantAlerted:      []bool{false},
		},
		{
			name:             "single-target-alert",
			failureThreshold: 0.5,
			targets:          []string{"target1"},
			total:            [][]int64{{1, 2, 3}},
			success:          [][]int64{{1, 2, 2}}, // Success didn't increase.
			wantAlerted:      []bool{true},
			wantAlerts:       []*AlertInfo{{Target: endpoint.Endpoint{Name: "target1"}, FailureRatio: 1.0, FailureThreshold: 0.5, FailingSince: time.Time{}.Add(2 * time.Second)}},
		},
		{
			name:              "duration-threshold-target2-alert",
			failureThreshold:  0.5,
			durationThreshold: time.Second,
			targets:           []string{"target1", "target2"},
			total:             [][]int64{{1, 2, 3, 4}, {1, 2, 3, 4}},
			success:           [][]int64{{1, 2, 2, 3}, {1, 2, 2, 2}},
			wantAlerted:       []bool{false, true},
			wantAlerts:        []*AlertInfo{{Target: endpoint.Endpoint{Name: "target2"}, FailureRatio: 1.0, FailureThreshold: 0.5, FailingSince: time.Time{}.Add(2 * time.Second)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah := &AlertHandler{
				failureThreshold:  tt.failureThreshold,
				durationThreshold: tt.durationThreshold,
				targets:           make(map[string]*targetState),
				notifyCh:          make(chan *AlertInfo, 10),
			}

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
		wantErr   bool
	}{
		{
			name:      "simple",
			probeName: "test-probe",
			conf: &configpb.AlertConf{
				Name:             "test-alert",
				FailureThreshold: 0.5,
			},
			want: &AlertHandler{
				name:              "test-alert",
				probeName:         "test-probe",
				failureThreshold:  0.5,
				durationThreshold: time.Duration(0),
				targets:           make(map[string]*targetState),
			},
		},
		{
			name:      "no-alert-name",
			probeName: "test-probe",
			conf: &configpb.AlertConf{
				FailureThreshold: 0.5,
			},
			want: &AlertHandler{
				name:              "test-probe",
				probeName:         "test-probe",
				failureThreshold:  0.5,
				durationThreshold: time.Duration(0),
				targets:           make(map[string]*targetState),
			},
		},
		{
			name:      "error-no-threshold",
			probeName: "test-probe",
			conf:      &configpb.AlertConf{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAlertHandler(tt.conf, tt.probeName, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAlertHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
