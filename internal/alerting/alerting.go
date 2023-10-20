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

// Package alerting implements alerting functionality in Cloudprober.
package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	"github.com/cloudprober/cloudprober/internal/alerting/notifier"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type targetState struct {
	lastSuccess int64
	lastTotal   int64
	failures    []bool

	alerted      bool
	alertTS      time.Time
	failingSince time.Time
}

// AlertHandler is responsible for handling alerts. It keeps track of the
// health of targets and notifies the user if there is a failure.
type AlertHandler struct {
	c            *configpb.AlertConf
	name         string
	probeName    string
	condition    *configpb.Condition
	notifyConfig *configpb.NotifyConfig
	notifyCh     chan *alertinfo.AlertInfo // Used only for testing for now.
	notifier     *notifier.Notifier

	mu      sync.Mutex
	targets map[string]*targetState
	l       *logger.Logger
}

// processConfig processes the alerting config and returns the updated config.
func processConfig(conf *configpb.AlertConf) {
	// Set default condition to 1 failure in 1 probe interval.
	if conf.GetCondition() == nil {
		conf.Condition = &configpb.Condition{
			Failures: 1,
			Total:    1,
		}
	}
	if conf.GetCondition().Total == 0 {
		conf.Condition.Total = conf.Condition.Failures
	}

	// Set default repeat interval to 1 hour.
	if conf.RepeatIntervalSec == nil {
		conf.RepeatIntervalSec = proto.Int32(3600)
	}
}

// NewAlertHandler creates a new AlertHandler from the given config.
// If the config is invalid, an error is returned.
func NewAlertHandler(conf *configpb.AlertConf, probeName string, l *logger.Logger) (*AlertHandler, error) {
	processConfig(conf)

	ah := &AlertHandler{
		c:            conf,
		name:         conf.GetName(),
		probeName:    probeName,
		condition:    conf.GetCondition(),
		notifyConfig: conf.GetNotify(),
		targets:      make(map[string]*targetState),
		l:            l,
	}

	if ah.name == "" {
		ah.name = probeName
	}

	// Initialize notifier.
	notifier, err := notifier.New(ah.c, l)
	if err != nil {
		return nil, fmt.Errorf("error creating notifier for alert %s: %v", ah.name, err)
	}
	ah.notifier = notifier

	return ah, nil
}

// extractValue is used to extract the total and success metric from an EventMetrics
// object.
func extractValue(em *metrics.EventMetrics, name string) (int64, error) {
	val := em.Metric(name)
	if val == nil {
		return 0, fmt.Errorf("%s metric not found in EventMetrics: %s", name, em.String())
	}

	numV, ok := val.(metrics.NumValue)
	if !ok {
		return 0, fmt.Errorf("%s metric doesn't have a numerical value: %s", name, numV.String())
	}
	return numV.Int64(), nil
}

func conditionID(alertKey string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(alertKey)).String()
}

func (ah *AlertHandler) notify(ep endpoint.Endpoint, ts *targetState, totalFailures int) {
	ah.l.Warningf("ALERT (%s): target (%s), failures (%d) higher than (%d) since (%v)", ah.name, ep.Name, totalFailures, ah.condition.Failures, ts.failingSince)

	ts.alerted = true
	alertKey := ah.globalKey(ep)

	alertInfo := &alertinfo.AlertInfo{
		Name:            ah.name,
		ProbeName:       ah.probeName,
		DeduplicationID: conditionID(alertKey),
		Target:          ep,
		Failures:        totalFailures,
		Total:           int(ah.condition.Total),
		FailingSince:    ts.failingSince,
	}

	if ah.notifyCh != nil {
		ah.notifyCh <- alertInfo
	}

	ah.notifier.Notify(context.Background(), alertInfo)
	globalState.add(alertKey, alertInfo)
}

func (ah *AlertHandler) resolveAlertCondition(ts *targetState, ep endpoint.Endpoint) {
	ah.l.Infof("ALERT Resolved (%s): target: %s", ah.name, ep.Name)

	ts.alerted = false
	ts.alertTS = time.Time{}

	key := ah.globalKey(ep)
	ai := globalState.get(key)
	if ai == nil {
		ah.l.Errorf("ALERT Resolved (%s): didn't find alert for target (%s) in the global state, will not send resolve notification", ah.name, ep.Name)
		return
	}

	ah.notifier.NotifyResolve(context.Background(), ai)
	globalState.resolve(key)
}

// handleAlertCondition handles the alert condition.
func (ah *AlertHandler) handleAlertCondition(ts *targetState, ep endpoint.Endpoint, timestamp time.Time, totalFailures int) {
	// Ongoing alert. Notify if the repeat interval has passed.
	if ts.alerted {
		if time.Since(ts.alertTS) > time.Duration(ah.c.GetRepeatIntervalSec())*time.Second {
			ts.alertTS = time.Now()
			ah.notify(ep, ts, totalFailures)
		}
		return
	}

	// New alert.
	ts.alerted = true
	ts.failingSince = timestamp
	ts.alertTS = time.Now()
	ah.notify(ep, ts, totalFailures)
}

func (ah *AlertHandler) globalKey(ep endpoint.Endpoint) string {
	return fmt.Sprintf("%s-%s-%s", ah.name, ah.probeName, ep.Key())
}

func (ah *AlertHandler) Record(ep endpoint.Endpoint, em *metrics.EventMetrics) {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	total, err := extractValue(em, "total")
	if err != nil {
		ah.l.ErrorAttrs(err.Error(), slog.String("target", ep.Name))
		return
	}
	success, err := extractValue(em, "success")
	if err != nil {
		ah.l.ErrorAttrs(err.Error(), slog.String("target", ep.Name))
		return
	}

	key := ep.Key()
	ts := ah.targets[key]
	if ts == nil {
		// If this is the very first probe for this target, we don't have
		// enough data to determine if it's failing or not. We just initialize
		// the target state and return.
		ts = &targetState{
			failures:    make([]bool, ah.condition.Total),
			lastTotal:   total,
			lastSuccess: success,
		}
		ah.targets[key] = ts
		return
	}

	totalCnt := int(total - ts.lastTotal)
	successCnt := int(success - ts.lastSuccess)
	if successCnt > totalCnt { // This should never happen.
		successCnt = totalCnt
	}

	// If totalCnt is negative, it means that the probe for this target was
	// reset for some reason. This should really never happen though.
	if totalCnt < 0 {
		ts.lastTotal = total
		ts.lastSuccess = success
		return
	}

	// If totalCnt is greater than the configured total, we only consider the
	// last ah.condition.Total samples.
	if totalCnt > int(ah.condition.Total) {
		excess := totalCnt - int(ah.condition.Total)
		totalCnt = int(ah.condition.Total)
		// To be safe, trim only successful samples from the data.
		successCnt = successCnt - excess
	}

	// Shift failures slice to the left by totalCnt. Note there will be no
	// shifting if len(ts.failures) <= total samples (default config case).
	leftIndex, rightIndex := 0, totalCnt
	for rightIndex < len(ts.failures) {
		ts.failures[leftIndex] = ts.failures[rightIndex]
		leftIndex++
		rightIndex++
	}

	// Populate recent fields in ts.failures based on the failures in the
	// current EventMetrics.
	failureCnt := totalCnt - successCnt
	for i := len(ts.failures) - 1; i > len(ts.failures)-totalCnt-1; i-- {
		ts.failures[i] = failureCnt > 0
		failureCnt--
	}

	totalFailures := 0
	for _, failed := range ts.failures {
		if failed {
			totalFailures++
		}
	}

	if totalFailures >= int(ah.condition.Failures) {
		ah.handleAlertCondition(ts, ep, em.Timestamp, totalFailures)
	} else if ts.alerted {
		ah.resolveAlertCondition(ts, ep)
	}

	ts.lastTotal, ts.lastSuccess = total, success
}
