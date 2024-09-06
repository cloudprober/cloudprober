// Copyright 2017-2024 The Cloudprober Authors.
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

package payload

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/itchyny/gojq"
)

type jsonMetric struct {
	metricsJQ  *gojq.Query // JQ filter to extract metrics map from JSON.
	metricName string
	valueJQ    *gojq.Query
}

type jsonMetricGroup struct {
	metrics  []*jsonMetric
	labelsJQ *gojq.Query
}

func sortedKeys[T any](m map[string]T) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func parseJSONMetricConfig(configs []*configpb.JSONMetric) ([]*jsonMetricGroup, error) {
	// Group configs by labelsJQFilter. Each group is exported as a separate
	// EventMetrics.
	cfgGroups := make(map[string][]*configpb.JSONMetric)
	for _, cfg := range configs {
		cfgGroups[cfg.GetLabelsJqFilter()] = append(cfgGroups[cfg.GetLabelsJqFilter()], cfg)
	}

	var out []*jsonMetricGroup
	for _, labelJQFilter := range sortedKeys(cfgGroups) {
		configs := cfgGroups[labelJQFilter]
		jmGroup := &jsonMetricGroup{}

		if labelJQFilter != "" {
			labelJQ, err := gojq.Parse(labelJQFilter)
			if err != nil {
				return nil, fmt.Errorf("error parsing labels_jq_filter: %v", err)
			}
			jmGroup.labelsJQ = labelJQ
		}

		for _, cfg := range configs {
			jm, err := parseJSONMetric(cfg)
			if err != nil {
				return nil, err
			}
			jmGroup.metrics = append(jmGroup.metrics, jm)
		}
		out = append(out, jmGroup)
	}
	return out, nil
}

func parseJSONMetric(cfg *configpb.JSONMetric) (*jsonMetric, error) {
	jm := &jsonMetric{}
	if cfg.GetMetricsJqFilter() != "" {
		if cfg.GetMetricName() != "" || cfg.GetValueJqFilter() != "" {
			return nil, fmt.Errorf("metrics_jq_filter is incompatible with metric_name and value_jq_filter")
		}
		metricsJQ, err := gojq.Parse(cfg.GetMetricsJqFilter())
		if err != nil {
			return nil, fmt.Errorf("error parsing metrics_jq_filter: %v", err)
		}
		jm.metricsJQ = metricsJQ
	} else {
		if cfg.GetMetricName() == "" || cfg.GetValueJqFilter() == "" {
			return nil, fmt.Errorf("either metrics_jq_filter or metric_name and value_jq_filter must be specified")
		}
		jm.metricName = cfg.GetMetricName()
		valueJQ, err := gojq.Parse(cfg.GetValueJqFilter())
		if err != nil {
			return nil, fmt.Errorf("error parsing value_jq_filter: %v", err)
		}
		jm.valueJQ = valueJQ
	}
	return jm, nil
}

func runJQFilter(jq *gojq.Query, input any) (any, error) {
	iter := jq.Run(input)
	var ret any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("jq query error: %v", err)
		}
		ret = v
	}
	return ret, nil
}

func jqValToMetricValue(v any) (metrics.Value, error) {
	switch v := v.(type) {
	case string:
		return metrics.NewString(v), nil
	case float64:
		return metrics.NewFloat(v), nil
	case bool:
		return metrics.NewInt(map[bool]int64{true: 1, false: 0}[v]), nil
	default:
		return nil, fmt.Errorf("jqValToMetricValue: unexpected type %T", v)
	}
}

func (jm *jsonMetric) process(input any, em *metrics.EventMetrics) error {
	if jm.metricsJQ != nil {
		metrics, err := runJQFilter(jm.metricsJQ, input)
		if err != nil {
			return err
		}
		metricsMap, ok := metrics.(map[string]any)
		if !ok {
			return fmt.Errorf("metrics_jq_filter didn't return a map[string]any: %v", metrics)
		}
		for _, k := range sortedKeys(metricsMap) {
			v, err := jqValToMetricValue(metricsMap[k])
			if err != nil {
				return err
			}
			em.AddMetric(k, v)
		}
		return nil
	}
	val, err := runJQFilter(jm.valueJQ, input)
	if err != nil {
		return err
	}
	v, err := jqValToMetricValue(val)
	if err != nil {
		return err
	}
	em.AddMetric(jm.metricName, v)
	return nil
}

func (jmGroup *jsonMetricGroup) process(input any) (*metrics.EventMetrics, error) {
	em := metrics.NewEventMetrics(time.Now())

	if jmGroup.labelsJQ != nil {
		labels, err := runJQFilter(jmGroup.labelsJQ, input)
		if err != nil {
			return nil, err
		}
		labelsMap, ok := labels.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("labels_jq_filter didn't return a map[string]any: %T", labels)
		}
		for k, v := range labelsMap {
			em.AddLabel(k, fmt.Sprintf("%v", v))
		}
	}

	for _, jm := range jmGroup.metrics {
		err := jm.process(input, em)
		if err != nil {
			return nil, err
		}
	}
	return em, nil
}

func (p *Parser) processJSONMetric(text []byte) []*metrics.EventMetrics {
	var input any
	err := json.Unmarshal(text, &input)
	if err != nil {
		p.l.Warningf("JSON validation failure: response %s is not a valid JSON", string(text))
		return nil
	}

	var ems []*metrics.EventMetrics

	for _, jmGroup := range p.jmGroups {
		em, err := jmGroup.process(input)
		if err != nil {
			p.l.Warning(err.Error())
			continue
		}
		ems = append(ems, em)
	}

	return ems
}
