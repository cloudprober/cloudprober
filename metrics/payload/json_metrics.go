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

func parseJSONMetricConfig(cfg *configpb.JSONMetric) (*jsonMetric, error) {
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

func (p *Parser) processJSONMetric(text []byte) *metrics.EventMetrics {
	var input any
	err := json.Unmarshal(text, &input)
	if err != nil {
		p.l.Warningf("JSON validation failure: response %s is not a valid JSON", string(text))
		return nil
	}

	em := metrics.NewEventMetrics(time.Now())
	for _, jm := range p.jsonMetrics {
		metricName, err := jm.getName(input)
		if err != nil {
			p.l.Warning(err.Error())
			continue
		}

		val, err := runJQFilter(jm.valueJQ, input)
		if err != nil {
			p.l.Warning(err.Error())
			continue
		}

		switch v := val.(type) {
		case string:
			em.AddMetric(metricName, metrics.NewString(v))
		case float64:
			em.AddMetric(metricName, metrics.NewFloat(v))
		case bool:
			em.AddMetric(metricName, metrics.NewInt(map[bool]int64{true: 1, false: 0}[v]))
		default:
			p.l.Warningf("processJSONMetric: unexpected type %T", val)
			continue
		}
	}

	return em
}
