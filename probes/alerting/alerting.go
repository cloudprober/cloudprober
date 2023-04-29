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
	"fmt"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

type targetState struct {
	lastSuccess  int64
	lastTotal    int64
	failures     []float32
	firstFailure time.Time
	alerted      bool
}

func (ts *targetState) reset() {
	ts.failures = ts.failures[:0]
	ts.firstFailure = time.Time{}
	ts.alerted = false
}

// AlertHandler is responsible for handling alerts. It keeps track of the
// health of targets and notifies the user if there is a failure.
type AlertHandler struct {
	name              string
	probeName         string
	failureThreshold  float32
	durationThreshold time.Duration
	notifyConfig      *configpb.Notify
	notifyCh          chan *AlertInfo // Used only for testing for now.

	mu      sync.Mutex
	targets map[string]*targetState
	l       *logger.Logger
}

// NewAlertHandler creates a new AlertHandler from the given config.
// If the config is invalid, an error is returned.
func NewAlertHandler(conf *configpb.AlertConf, probeName string, l *logger.Logger) (*AlertHandler, error) {
	if conf.GetFailureThreshold() == 0 {
		return nil, fmt.Errorf("invalid failure_threshold (0)")
	}

	name := conf.GetName()
	if name == "" {
		name = probeName
	}

	return &AlertHandler{
		name:              name,
		probeName:         probeName,
		failureThreshold:  conf.GetFailureThreshold(),
		durationThreshold: time.Second * time.Duration(conf.GetDurationThresholdSec()),
		notifyConfig:      conf.GetNotify(),
		targets:           make(map[string]*targetState),
		l:                 l,
	}, nil
}

// extractValues is used to extract the total and success metric from an EventMetrics
// object.
func extractValues(em *metrics.EventMetrics) (int64, int64, error) {
	successM, totalM := em.Metric("success"), em.Metric("total")

	numV, ok := totalM.(metrics.NumValue)
	if !ok {
		return 0, 0, fmt.Errorf("total metric doesn't have a numerical value: %s", numV.String())
	}
	total := numV.Int64()

	numV, ok = successM.(metrics.NumValue)
	if !ok {
		return 0, 0, fmt.Errorf("success metric doesn't have a numerical value: %s", numV.String())
	}
	success := numV.Int64()

	return total, success, nil
}

func (ah *AlertHandler) Record(ep endpoint.Endpoint, em *metrics.EventMetrics) error {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	total, success, err := extractValues(em)
	if err != nil {
		return err
	}

	key := ep.Key()

	ts := ah.targets[key]
	if ts == nil {
		ts = &targetState{}
		ah.targets[key] = ts
	}

	// First time or a reset
	if ts.lastTotal == 0 || total < ts.lastTotal {
		ts.lastTotal, ts.lastSuccess = total, success
		return nil
	}

	failureLastInterval := (total - ts.lastTotal) - (success - ts.lastSuccess)
	failureRatio := float32(failureLastInterval) / float32((total - ts.lastTotal))

	if failureRatio > ah.failureThreshold {
		if ts.firstFailure.IsZero() {
			ts.firstFailure = em.Timestamp
		}
		ts.failures = append(ts.failures, failureRatio)

		if em.Timestamp.Sub(ts.firstFailure) >= ah.durationThreshold {
			if !ts.alerted {
				ah.notify(ep, ts, failureRatio)
			}
		}
	} else {
		ts.reset()
	}

	ts.lastTotal, ts.lastSuccess = total, success
	return nil
}
