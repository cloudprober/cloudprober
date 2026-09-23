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
	"math/big"
	"sort"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/itchyny/gojq"
)

type jsonMetric struct {
	metricsJQ *gojq.Query // JQ filter to extract metrics map from JSON.
	labelsJQ  *gojq.Query
}

func sortedKeys[T any](m map[string]T) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func parseJSONMetricConfig(configs []*configpb.JSONMetric) ([]*jsonMetric, error) {
	var out []*jsonMetric
	for _, cfg := range configs {
		jm := &jsonMetric{}

		metricsJQ, err := gojq.Parse(cfg.GetJqFilter())
		if err != nil {
			return nil, fmt.Errorf("error parsing jq_filter: %v", err)
		}
		jm.metricsJQ = metricsJQ

		if cfg.GetLabelsJqFilter() != "" {
			labelJQ, err := gojq.Parse(cfg.GetLabelsJqFilter())
			if err != nil {
				return nil, fmt.Errorf("error parsing labels_jq_filter: %v", err)
			}
			jm.labelsJQ = labelJQ
		}
		out = append(out, jm)
	}
	return out, nil
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
	// gojq returns integer literals and integer-valued results (e.g. length,
	// add) as int, and *big.Int for values that overflow int.
	case int:
		return metrics.NewInt(int64(v)), nil
	case *big.Int:
		// metrics.Int is int64-backed; values that overflow it fall back to
		// float, matching how large JSON numbers decode (via float64).
		if v.IsInt64() {
			return metrics.NewInt(v.Int64()), nil
		}
		f, _ := new(big.Float).SetInt(v).Float64()
		return metrics.NewFloat(f), nil
	case bool:
		return metrics.NewInt(map[bool]int64{true: 1, false: 0}[v]), nil
	default:
		return nil, fmt.Errorf("unexpected value (%v) type %T", v, v)
	}
}

func (jm *jsonMetric) process(input any, em *metrics.EventMetrics) (*metrics.EventMetrics, error) {
	metrics, err := runJQFilter(jm.metricsJQ, input)
	if err != nil {
		return nil, err
	}

	metricsMap, ok := metrics.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("jq_filter didn't return a map[string]any: %v", metrics)
	}
	for _, k := range sortedKeys(metricsMap) {
		v, err := jqValToMetricValue(metricsMap[k])
		if err != nil {
			return nil, err
		}
		em.AddMetric(k, v)
	}

	if jm.labelsJQ != nil {
		labels, err := runJQFilter(jm.labelsJQ, input)
		if err != nil {
			return nil, err
		}
		labelsMap, ok := labels.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("labels_jq_filter didn't return a map[string]any: %T", labels)
		}
		for _, k := range sortedKeys(labelsMap) {
			em.AddLabel(k, fmt.Sprintf("%v", labelsMap[k]))
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

	for _, jm := range p.jsonMetrics {
		em, err := jm.process(input, p.newEM(nil))
		if err != nil {
			p.l.Warning(err.Error())
			continue
		}
		ems = append(ems, em)
	}

	return ems
}
