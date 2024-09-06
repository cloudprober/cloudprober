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
	metricName string
	nameJQ     *gojq.Query
	valueJQ    *gojq.Query
}

type jsonMetricGroup struct {
	metrics  []*jsonMetric
	labelsJQ *gojq.Query
}

func parseJSONMetricConfig(configs []*configpb.JSONMetric) ([]*jsonMetricGroup, error) {
	// Group configs by labelsJQFilter. Each group is exported as a separate
	// EventMetrics.
	var keys []string
	cfgGroups := make(map[string][]*configpb.JSONMetric)
	for _, cfg := range configs {
		cfgGroups[cfg.GetLabelsJqFilter()] = append(cfgGroups[cfg.GetLabelsJqFilter()], cfg)
	}
	for k := range cfgGroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []*jsonMetricGroup
	for _, labelJQFilter := range keys {
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
	jm := &jsonMetric{metricName: cfg.GetMetricName()}
	if cfg.GetNameJqFilter() != "" {
		nameJQ, err := gojq.Parse(cfg.GetNameJqFilter())
		if err != nil {
			return nil, fmt.Errorf("error parsing name_jq_filter: %v", err)
		}
		jm.nameJQ = nameJQ
	}
	if jm.metricName == "" && jm.nameJQ == nil {
		return nil, fmt.Errorf("either name or name_jq_filter must be specified")
	}

	if cfg.GetValueJqFilter() == "" {
		return nil, fmt.Errorf("value_jq_filter must be specified")
	}
	valueJQ, err := gojq.Parse(cfg.GetValueJqFilter())
	if err != nil {
		return nil, fmt.Errorf("error parsing name_jq_filter: %v", err)
	}
	jm.valueJQ = valueJQ
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

func (jm *jsonMetric) getName(input any) (string, error) {
	if jm.metricName != "" {
		return jm.metricName, nil
	}

	name, err := runJQFilter(jm.nameJQ, input)
	if err != nil {
		return "", err
	}
	s, ok := name.(string)
	if !ok {
		return "", fmt.Errorf("name_jq_filter didn't return a string: %T", name)
	}
	return s, nil
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
		metricName, err := jm.getName(input)
		if err != nil {
			return nil, err
		}

		val, err := runJQFilter(jm.valueJQ, input)
		if err != nil {
			return nil, err
		}

		switch v := val.(type) {
		case string:
			em.AddMetric(metricName, metrics.NewString(v))
		case float64:
			em.AddMetric(metricName, metrics.NewFloat(v))
		case bool:
			em.AddMetric(metricName, metrics.NewInt(map[bool]int64{true: 1, false: 0}[v]))
		default:
			return nil, fmt.Errorf("processJSONMetric: unexpected type %T", val)
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
