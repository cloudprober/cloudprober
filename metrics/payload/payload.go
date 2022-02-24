// Copyright 2017-2022 The Cloudprober Authors.
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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
)

// Parser encapsulates the config for parsing payloads to metrics.
type Parser struct {
	baseEM      *metrics.EventMetrics
	distMetrics map[string]*metrics.Distribution
	aggMetrics  map[string]*metrics.EventMetrics
	aggregate   bool
	l           *logger.Logger
}

// NewParser returns a new payload parser, based on the config provided.
func NewParser(opts *configpb.OutputMetricsOptions, ptype, probeName string, defaultKind metrics.Kind, l *logger.Logger) (*Parser, error) {
	parser := &Parser{
		aggregate:   opts.GetAggregateInCloudprober(),
		distMetrics: make(map[string]*metrics.Distribution),
		aggMetrics:  make(map[string]*metrics.EventMetrics),
		l:           l,
	}

	// If there are any distribution metrics, build them now itself.
	for name, distMetric := range opts.GetDistMetric() {
		d, err := metrics.NewDistributionFromProto(distMetric)
		if err != nil {
			return nil, err
		}
		parser.distMetrics[name] = d
	}

	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", ptype).
		AddLabel("probe", probeName)

	switch opts.GetMetricsKind() {
	case configpb.OutputMetricsOptions_CUMULATIVE:
		em.Kind = metrics.CUMULATIVE
	case configpb.OutputMetricsOptions_GAUGE:
		if opts.GetAggregateInCloudprober() {
			return nil, errors.New("payload.NewParser: invalid config, GAUGE metrics should not have aggregate_in_cloudprober enabled")
		}
		em.Kind = metrics.GAUGE
	case configpb.OutputMetricsOptions_UNDEFINED:
		em.Kind = defaultKind
	}

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

func updateMetricValue(mv metrics.Value, val string) error {
	// If a distribution, process it through processDistValue.
	if mVal, ok := mv.(*metrics.Distribution); ok {
		if err := processDistValue(mVal, val); err != nil {
			return fmt.Errorf("error parsing distribution value (%s): %v", val, err)
		}
		return nil
	}

	v, err := metrics.ParseValueFromString(val)
	if err != nil {
		return fmt.Errorf("error parsing value (%s): %v", val, err)
	}

	return mv.Add(v)
}

func parseLabels(labelStr string) [][2]string {
	var labels [][2]string
	for _, l := range strings.Split(labelStr, ",") {
		parts := strings.SplitN(strings.TrimSpace(l), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		// Unquote val if it is a quoted string. strconv returns an error if string
		// is not quoted at all or is unproperly quoted. We use raw string in that
		// case.
		uval, err := strconv.Unquote(val)
		if err == nil {
			val = uval
		}
		labels = append(labels, [2]string{key, val})
	}
	return labels
}

func parseLine(line string) (string, string, string, error) {
	ob := strings.Index(line, "{")
	// If "{" was not found or was the last element, assume label-less metric.

	if ob == -1 || ob == len(line)-1 {
		// Parse line as metric has no labels.
		varKV := strings.SplitN(line, " ", 2)
		if len(varKV) < 2 {
			return "", "", "", fmt.Errorf("wrong var key-value format: %s", line)
		}
		return varKV[0], "", strings.TrimSpace(varKV[1]), nil
	}

	// Capture metric name and move line-beginning forward.
	metricName := line[:ob]
	line = line[ob+1:]

	eb := strings.Index(line, "}")
	// If "}" was not found or was the last element, invalid line.
	if eb == -1 || eb == len(line)-1 {
		return "", "", "", fmt.Errorf("invalid line (%s), only opening brace found", line)
	}

	// Capture label string and move line-beginning forward.
	labelStr := line[:eb]
	line = line[eb+1:]

	return metricName, labelStr, strings.TrimSpace(line), nil
}

func (p *Parser) metricValueLabels(line string) (metricName, val string, labels [][2]string, err error) {
	metricName, labelStr, value, err := parseLine(line)
	if err != nil {
		return "", "", nil, err
	}

	return metricName, value, parseLabels(labelStr), nil
}

func addNewMetric(em *metrics.EventMetrics, metricName, val string) error {
	// New metric name, make sure it's not disallowed.
	switch metricName {
	case "success", "total", "latency":
		return fmt.Errorf("metric name (%s) in the payload conflicts with standard metrics: (success,total,latency), ignoring", metricName)
	}

	v, err := metrics.ParseValueFromString(val)
	if err != nil {
		return fmt.Errorf("could not parse value (%s) for the new metric name (%s): %v", val, metricName, err)
	}

	em.AddMetric(metricName, v)
	return nil
}

func (p *Parser) newEM(ts time.Time, target, metricName, val string, labels [][2]string) (*metrics.EventMetrics, error) {
	em := p.baseEM.Clone().AddLabel("dst", target)
	em.Timestamp = ts
	for _, kv := range labels {
		em.AddLabel(kv[0], kv[1])
	}

	// If it's a pre-configured, distribution metric.
	if dv, ok := p.distMetrics[metricName]; ok {
		d := dv.Clone().(*metrics.Distribution)
		if err := processDistValue(d, val); err != nil {
			return nil, err
		}
		em.AddMetric(metricName, d)
		return em, nil
	}

	if err := addNewMetric(em, metricName, val); err != nil {
		return nil, err
	}

	return em, nil
}

func metricKey(name, target string, labels [][2]string) string {
	var parts []string
	parts = append(parts, name)
	for _, l := range append(labels, [2]string{"dst", target}) {
		parts = append(parts, fmt.Sprintf("%s=%s", l[0], l[1]))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// payloadLineMetrics parses a payload line, and either updates an existing
// EventMetrics(EM), or creates a new one.
func (p *Parser) payloadLineMetrics(payloadTS time.Time, line, target string) (*metrics.EventMetrics, error) {
	metricName, val, labels, err := p.metricValueLabels(line)
	if err != nil {
		return nil, fmt.Errorf("error while parsing line (%s): %v", line, err)
	}

	// Non-aggregate case is straightforward. Just build an EM and return.
	if !p.aggregate {
		em, err := p.newEM(payloadTS, target, metricName, val, labels)
		if err != nil {
			return nil, err
		}
		return em, nil
	}

	// If aggregating, create a key, find if we already have an EM with that key
	// if yes, update that metric, or create a new metric.
	key := metricKey(metricName, target, labels)

	if em := p.aggMetrics[key]; em != nil {
		if err := updateMetricValue(em.Metric(metricName), val); err != nil {
			return nil, fmt.Errorf("error updating metric %s with val %s: %v", metricName, val, err)
		}
		em.Timestamp = payloadTS
		return em.Clone(), nil
	}

	em, err := p.newEM(payloadTS, target, metricName, val, labels)
	if err != nil {
		return nil, err
	}

	p.aggMetrics[key] = em
	return em.Clone(), nil
}

// PayloadMetrics parses the given payload and creates one EventMetrics per
// line. Each metric line can have its own labels, e.g. num_rows{db=dbA}.
func (p *Parser) PayloadMetrics(payload, target string) []*metrics.EventMetrics {
	payloadTS := time.Now()
	var results []*metrics.EventMetrics
	for _, line := range strings.Split(payload, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		em, err := p.payloadLineMetrics(payloadTS, line, target)
		if err != nil {
			p.l.Warning(err.Error())
			continue
		}
		results = append(results, em)
	}
	return results
}

// processDistValue processes a distribution value. It works with distribution
// values in 2 formats:
// a) a full distribution string, capturing all the details, e.g.
//    "dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12"
// a) a comma-separated list of floats, where distribution details have been
//    provided at the time of config, e.g.
//    "12,13,10.1,9.875,11.1"
func processDistValue(mVal *metrics.Distribution, val string) error {
	if val[0] == 'd' {
		distVal, err := metrics.ParseDistFromString(val)
		if err != nil {
			return err
		}
		return mVal.Add(distVal)
	}

	// It's a pre-defined distribution metric
	for _, s := range strings.Split(val, ",") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("unsupported value for distribution metric (expected comma separated list of float64s): %s", val)
		}
		mVal.AddFloat64(f)
	}
	return nil
}
