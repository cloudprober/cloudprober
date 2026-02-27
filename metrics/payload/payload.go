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

// Package payload provides utilities to work with the metrics in payload.
package payload

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
)

func emKind(k configpb.OutputMetricsOptions_MetricsKind) metrics.Kind {
	switch k {
	case configpb.OutputMetricsOptions_CUMULATIVE:
		return metrics.CUMULATIVE
	case configpb.OutputMetricsOptions_GAUGE:
		return metrics.GAUGE
	default:
		return metrics.GAUGE
	}
}

// Parser encapsulates the config for parsing payloads to metrics.
type Parser struct {
	opts              *configpb.OutputMetricsOptions
	baseEM            *metrics.EventMetrics
	distMetrics       map[string]*metrics.Distribution
	aggregatedMetrics map[string]*metrics.EventMetrics
	aggregate         bool
	mu                sync.Mutex

	jsonMetrics []*jsonMetric

	lineAcceptRe *regexp.Regexp
	lineRejectRe *regexp.Regexp

	l *logger.Logger
}

type Input struct {
	Response interface{}
	Text     []byte
}

func (p *Parser) newEM(labels [][2]string) *metrics.EventMetrics {
	em := p.baseEM.Clone()
	for _, kv := range labels {
		em.AddLabel(kv[0], kv[1])
	}
	return em
}

// NewParser returns a new payload parser, based on the config provided.
func NewParser(opts *configpb.OutputMetricsOptions, l *logger.Logger) (*Parser, error) {
	parser := &Parser{
		opts:              opts,
		aggregate:         opts.GetAggregateInCloudprober(),
		distMetrics:       make(map[string]*metrics.Distribution),
		aggregatedMetrics: make(map[string]*metrics.EventMetrics),
		l:                 l,
	}

	if opts.GetLineAcceptRegex() != "" {
		re, err := regexp.Compile(opts.GetLineAcceptRegex())
		if err != nil {
			return nil, fmt.Errorf("payload.NewParser: error compiling line accept regex: %v", err)
		}
		parser.lineAcceptRe = re
	}

	if opts.GetLineRejectRegex() != "" {
		re, err := regexp.Compile(opts.GetLineRejectRegex())
		if err != nil {
			return nil, fmt.Errorf("payload.NewParser: error compiling line reject regex: %v", err)
		}
		parser.lineRejectRe = re
	}

	jsonMetricGroups, err := parseJSONMetricConfig(opts.GetJsonMetric())
	if err != nil {
		return nil, err
	}
	parser.jsonMetrics = jsonMetricGroups

	// If there are any distribution metrics, build them now itself.
	for name, distMetric := range opts.GetDistMetric() {
		d, err := metrics.NewDistributionFromProto(distMetric)
		if err != nil {
			return nil, err
		}
		parser.distMetrics[name] = d
	}

	em := metrics.NewEventMetrics(time.Now())
	em.Kind = emKind(opts.GetMetricsKind())

	// Labels are specified in the probe config.
	if opts.GetAdditionalLabels() != "" {
		for _, label := range strings.Split(opts.GetAdditionalLabels(), ",") {
			labelKV := strings.Split(label, "=")
			if len(labelKV) != 2 {
				return nil, fmt.Errorf("payload.NewParser: invalid config, wrong label format: %v", labelKV)
			}
			em.AddLabel(labelKV[0], labelKV[1])
		}
	}

	parser.baseEM = em

	return parser, nil
}

func withTimestampAndNoAlert(ems []*metrics.EventMetrics, ts time.Time) []*metrics.EventMetrics {
	for _, em := range ems {
		em.Timestamp = ts
		em.SetNotForAlerting()
	}
	return ems
}

// PayloadMetrics parses the given payload and creates one EventMetrics per
// line. Each metric line can have its own labels, e.g. num_rows{db=dbA}.
// Note: target is used to uniquely identify metrics for aggregation.
func (p *Parser) PayloadMetrics(input *Input, targetKey string) []*metrics.EventMetrics {
	ts := time.Now()
	if p.opts.GetHeaderMetric() == nil && p.opts.GetJsonMetric() == nil {
		return withTimestampAndNoAlert(p.lineBasedMetrics(input.Text, targetKey), ts)
	}

	var results []*metrics.EventMetrics
	if p.opts.GetHeaderMetric() != nil {
		results = append(results, p.processHeaderMetrics(input.Response))
	}

	if p.opts.GetJsonMetric() != nil {
		results = append(results, p.processJSONMetric(input.Text)...)
	}

	return withTimestampAndNoAlert(results, ts)
}
