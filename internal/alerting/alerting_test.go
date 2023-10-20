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
	"github.com/cloudprober/cloudprober/internal/alerting/notifier"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func testAlertInfo(target string, failures, total, dur int) *alertinfo.AlertInfo {
	ep := endpoint.Endpoint{Name: target}
	ah := &AlertHandler{
		name:      "test-probe",
		probeName: "test-probe",
	}

	return &alertinfo.AlertInfo{
		Name:            ah.name,
		ProbeName:       ah.probeName,
		Target:          ep,
		Failures:        failures,
		Total:           total,
		FailingSince:    time.Time{}.Add(time.Duration(dur) * time.Second),
		DeduplicationID: conditionID(ah.globalKey(ep)),
	}
}

type testData struct {
	total, success []int64
}

type testAlertHandlerArgs struct {
	name        string
	condition   *configpb.Condition
	targets     map[string]testData
	wantAlerted map[string]bool
	wantAlerts  []*alertinfo.AlertInfo
	alertCfg    *configpb.AlertConf
	waitTime    time.Duration
}

func testAlertHandlerBehavior(t *testing.T, tt testAlertHandlerArgs) {
	t.Helper()

	if tt.alertCfg == nil {
		tt.alertCfg = &configpb.AlertConf{}
	}
	tt.alertCfg.Condition = tt.condition
	ah, err := NewAlertHandler(tt.alertCfg, "test-probe", nil)
	assert.NoError(t, err)

	ah.notifyCh = make(chan *alertinfo.AlertInfo, 10)

	for target, td := range tt.targets {
		ep := endpoint.Endpoint{Name: target}
		ts := time.Time{}

		for i := range td.total {
			em := metrics.NewEventMetrics(ts)
			em.AddMetric("total", metrics.NewInt(td.total[i]))
			em.AddMetric("success", metrics.NewInt(td.success[i]))

			ah.Record(ep, em)
			t.Logf("target (%s) state: %+v", target, ah.targets[ep.Key()])

			ts = ts.Add(time.Second)
			time.Sleep(tt.waitTime)
		}

		// Verify that target is in expected alerted state after the
		// run.
		assert.Equal(t, tt.wantAlerted[target], ah.targets[ep.Key()].alerted, target+" alerted")
	}

	// Verify that alerts are sent on the notify channel.
	assert.Equal(t, len(tt.wantAlerts), len(ah.notifyCh), "number of alerts")
	if len(tt.wantAlerts) == len(ah.notifyCh) {
		for i := range tt.wantAlerts {
			a := <-ah.notifyCh
			assert.Equal(t, tt.wantAlerts[i], a)
		}
	}
}

func TestAlertHandlerRecord(t *testing.T) {
	tests := []struct {
		name           string
		condition      *configpb.Condition
		total, success []int64
		wantAlerted    bool
		wantAlerts     []*alertinfo.AlertInfo
	}{
		{
			name:        "single-target-no-alert",
			total:       []int64{1, 2},
			success:     []int64{1, 2},
			wantAlerted: false,
		},
		{
			name:        "single-target-alert-default-condition",
			total:       []int64{1, 2, 3},
			success:     []int64{1, 2, 2}, // Success didn't increase.
			wantAlerted: true,
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 2)},
		},
		{
			name:        "default-condition-one-point-no-alert",
			total:       []int64{2},
			success:     []int64{1},
			wantAlerted: false,
		},
		{
			name:        "alerts-last-alert-cleared",
			total:       []int64{2, 4, 6, 8},
			success:     []int64{1, 3, 4, 6},
			wantAlerted: false,
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 2)},
		},
		{
			name:        "alert-over-a-period-of-time",
			condition:   &configpb.Condition{Failures: int32(3), Total: int32(5)},
			total:       []int64{2, 4, 6, 8}, // total: 2, 2, 2
			success:     []int64{1, 2, 4, 4}, // failures: 1, 0, 2
			wantAlerted: true,
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target1", 3, 5, 3)},
		},
		{
			name:        "over-a-period-of-time-alert-cleared",
			condition:   &configpb.Condition{Failures: int32(3), Total: int32(5)},
			total:       []int64{2, 4, 6, 8, 10}, // total: 2, 2, 2, 2
			success:     []int64{1, 2, 4, 4, 6},  // failures: 1, 0, 2, 0
			wantAlerted: false,
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target1", 3, 5, 3)},
		},
		{
			name:      "alert-cleared-and-alerted-again",
			condition: &configpb.Condition{Failures: int32(3), Total: int32(5)},
			total:     []int64{2, 4, 6, 8, 10, 12}, // total:    2, 2, 2, 2, 2
			success:   []int64{1, 2, 4, 4, 6, 6},   // failures: 1, 0, 2, 0, 2
			// total:    2, 2, 2, 2, 2
			// failures: 1, 0, 2, 0, 2
			wantAlerted: true,
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target1", 3, 5, 3), testAlertInfo("target1", 3, 5, 5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAlertHandlerBehavior(t, testAlertHandlerArgs{
				alertCfg:    &configpb.AlertConf{},
				condition:   tt.condition,
				targets:     map[string]testData{"target1": {total: tt.total, success: tt.success}},
				wantAlerted: map[string]bool{"target1": tt.wantAlerted},
				wantAlerts:  tt.wantAlerts,
			})
		})
	}
}

func TestNotificationRepeat(t *testing.T) {
	tests := []struct {
		name           string
		condition      *configpb.Condition
		alertCfg       *configpb.AlertConf
		total, success []int64
		waitTime       time.Duration
		wantAlerts     []*alertinfo.AlertInfo
	}{
		{
			name:       "continuous-condition-single-notification",
			alertCfg:   &configpb.AlertConf{},
			total:      []int64{1, 2, 3},
			success:    []int64{1, 1, 1},
			waitTime:   10 * time.Millisecond,
			wantAlerts: []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 1)},
		},
		{
			name:       "continuous-condition-repeat-notification",
			alertCfg:   &configpb.AlertConf{RepeatIntervalSec: proto.Int32(0)},
			total:      []int64{1, 2, 3},
			success:    []int64{1, 1, 1},
			waitTime:   10 * time.Millisecond,
			wantAlerts: []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 1), testAlertInfo("target1", 1, 1, 1)},
		},
		{
			name:       "continuous-condition-no-repeat-yet",
			alertCfg:   &configpb.AlertConf{RepeatIntervalSec: proto.Int32(1)},
			total:      []int64{1, 2, 3, 4},
			success:    []int64{1, 1, 1, 1},
			waitTime:   10 * time.Millisecond,
			wantAlerts: []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 1)},
		},
		{
			name:       "continuous-condition-for-1-sec",
			alertCfg:   &configpb.AlertConf{RepeatIntervalSec: proto.Int32(1)},
			total:      []int64{1, 2, 3, 4, 5, 6, 7, 8},
			success:    []int64{1, 1, 1, 1, 1, 1, 1, 1},
			waitTime:   200 * time.Millisecond,
			wantAlerts: []*alertinfo.AlertInfo{testAlertInfo("target1", 1, 1, 1), testAlertInfo("target1", 1, 1, 1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAlertHandlerBehavior(t, testAlertHandlerArgs{
				condition:   tt.condition,
				alertCfg:    tt.alertCfg,
				targets:     map[string]testData{"target1": {total: tt.total, success: tt.success}},
				wantAlerted: map[string]bool{"target1": true},
				wantAlerts:  tt.wantAlerts,
				waitTime:    tt.waitTime,
			})
		})
	}
}

func TestAlertHandlerRecordTwoTargets(t *testing.T) {
	tests := []testAlertHandlerArgs{
		{
			name:      "only-target2-alert",
			condition: &configpb.Condition{Failures: int32(2), Total: int32(0)},
			targets: map[string]testData{
				"target1": {
					total:   []int64{1, 2, 3, 4}, // total: 1, 1, 1
					success: []int64{1, 2, 2, 3}, // failures: 0, 1, 0
				},
				"target2": {
					total:   []int64{1, 2, 3, 4}, // total: 1, 1, 1
					success: []int64{1, 2, 2, 2}, // failures: 0, 1, 1
				},
			},
			wantAlerted: map[string]bool{"target1": false, "target2": true},
			wantAlerts:  []*alertinfo.AlertInfo{testAlertInfo("target2", 2, 2, 3)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAlertHandlerBehavior(t, tt)
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
				c: &configpb.AlertConf{
					Name:              "test-alert",
					Condition:         &configpb.Condition{Failures: 1, Total: 1},
					RepeatIntervalSec: proto.Int32(3600),
				},
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
				c: &configpb.AlertConf{
					Condition:         &configpb.Condition{Failures: 4, Total: 5},
					RepeatIntervalSec: proto.Int32(3600),
				},
				notifier: &notifier.Notifier{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := notifier.New(tt.conf, nil)
			assert.NoError(t, err, "notifier.New(%v, nil) failed", tt.conf)
			tt.want.notifier = n
			ah, err := NewAlertHandler(tt.conf, tt.probeName, nil)
			assert.NoError(t, err, "NewAlertHandler(%v, %v, nil) failed", tt.conf, tt.probeName)
			assert.Equal(t, tt.want, ah)
		})
	}
}

func TestExtractValue(t *testing.T) {
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(2)).
		AddMetric("success", metrics.NewInt(1))

	tests := []struct {
		name    string
		want    int64
		wantErr bool
	}{
		{
			name: "total",
			want: 2,
		},
		{
			name: "success",
			want: 1,
		},
		{
			name:    "success-err",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractValue(em, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionID(t *testing.T) {
	tests := []struct {
		alertKey string
		want     string
	}{
		{
			alertKey: "test-probe-target1",
			want:     "33b0ac57-887d-3205-bba1-26f0d9a97104",
		},
		{
			alertKey: "test-probe-target1",
			want:     "33b0ac57-887d-3205-bba1-26f0d9a97104",
		},
		{
			alertKey: "test-probe-target2",
			want:     "e93e4809-4dc5-3a74-972c-a30d2c253e97",
		},
	}
	for _, tt := range tests {
		t.Run(tt.alertKey, func(t *testing.T) {
			assert.Equal(t, tt.want, conditionID(tt.alertKey))
		})
	}
}
