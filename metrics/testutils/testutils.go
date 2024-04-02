// Copyright 2019 The Cloudprober Authors.
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

/*
Package testutils provides utilities for tests.
*/
package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
)

// MetricsFromChannel reads metrics.EventMetrics from dataChannel with a timeout
func MetricsFromChannel(dataChan chan *metrics.EventMetrics, num int, timeout time.Duration) (results []*metrics.EventMetrics, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var timedout bool

	for i := 0; i < num && !timedout; i++ {
		select {
		case em := <-dataChan:
			results = append(results, em)
		case <-ctx.Done():
			timedout = true
		}
	}

	if timedout {
		err = fmt.Errorf("timed out while waiting for data from dataChannel, got only %d metrics out of %d", len(results), num)
	}
	return
}

func LabelsMapByTarget(ems []*metrics.EventMetrics) map[string]map[string]string {
	lmap := make(map[string]map[string]string)
	for _, em := range ems {
		target := em.Label("dst")
		if lmap[target] == nil {
			lmap[target] = make(map[string]string)
		}
		for _, k := range em.LabelsKeys() {
			lmap[target][k] = em.Label(k)
		}
	}
	return lmap
}

func EventMetricsByTargetMetric(ems []*metrics.EventMetrics) map[string]map[string][]*metrics.EventMetrics {
	emMap := make(map[string]map[string][]*metrics.EventMetrics)
	for _, em := range ems {
		target := em.Label("dst")
		if emMap[target] == nil {
			emMap[target] = make(map[string][]*metrics.EventMetrics)
		}
		for _, k := range em.MetricsKeys() {
			emMap[target][k] = append(emMap[target][k], em)
		}
	}
	return emMap
}

type MetricsMap map[string]map[string][]metrics.Value

// MetricsMapByTarget rearranges a list of metrics into a map of map.
//
//	{
//	  "target1": {
//	      "m1": [val1, val2..],
//	      "m2": [val1],
//	  },
//	  "target2": {
//	      ...
//	  }
//	}
func MetricsMapByTarget(ems []*metrics.EventMetrics) MetricsMap {
	mmap := make(map[string]map[string][]metrics.Value)
	for _, em := range ems {
		target := em.Label("dst")
		if mmap[target] == nil {
			mmap[target] = make(map[string][]metrics.Value)
		}
		for _, k := range em.MetricsKeys() {
			mmap[target][k] = append(mmap[target][k], em.Metric(k))
		}
	}
	return mmap
}

// Filter returns a map of target to values for a given metric name.
func (mmap MetricsMap) Filter(metricName string) map[string][]metrics.Value {
	res := make(map[string][]metrics.Value, len(mmap))
	for tgt, valMap := range mmap {
		for name, vals := range valMap {
			if name != metricName {
				continue
			}
			res[tgt] = vals
		}
	}
	return res
}

// LastValueInt64 returns the last value for a given metric name and target.
func (mmap MetricsMap) LastValueInt64(target, metricName string) int64 {
	for tgt, valMap := range mmap {
		if tgt != target {
			continue
		}
		for name, vals := range valMap {
			if name != metricName {
				continue
			}
			if len(vals) == 0 {
				return -1
			}
			return vals[len(vals)-1].(metrics.NumValue).Int64()
		}
	}
	return -1
}
