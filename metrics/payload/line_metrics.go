// Copyright 2017-2026 The Cloudprober Authors.
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
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/metrics"
)

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func readLabelKey(s string) (string, string, error) {
	s = strings.TrimLeft(s, " ")
	i := 0
	for i < len(s) && s[i] != '=' {
		if !isAlphanumeric(s[i]) && s[i] != '_' {
			return "", "", fmt.Errorf("invalid key char (%v), input: %s", s[i], s)
		}
		i++
	}
	if i == len(s) || s[i] != '=' {
		return "", "", fmt.Errorf("no value found in the input: %s", s)
	}
	return s[:i], s[i+1:], nil
}

func readLabelValue(s string) (string, string, error) {
	s = strings.TrimLeft(s, " ")
	if len(s) == 0 {
		return "", "", nil
	}

	// Verify that we either have zero remainder or it starts with a ','
	verifyReturn := func(val, rem string) (string, string, error) {
		if len(rem) == 0 {
			return val, rem, nil
		}
		if rem[0] != ',' {
			return "", "", fmt.Errorf("value doesn't end with a ',' and not final value: %s", rem)
		}
		return val, rem[1:], nil
	}

	if s[0] == '"' {
		// Quoted value
		s = s[1:]
		for i := 0; i < len(s); i++ {
			if s[i] == '\\' {
				i++ // skip next char
				continue
			}
			if s[i] == '"' {
				return verifyReturn(s[:i], s[i+1:])
			}
		}
		// Coming out of this loop means we don't find closing quote
		return "", "", fmt.Errorf("no closing quote found in the input: %s", s)
	}

	// Unquoted value, for unquoted value, we support only specific chars.
	i := 0
	for ; i < len(s) && s[i] != ','; i++ {
		if !isAlphanumeric(s[i]) && !slices.Contains([]byte{'-', '_', '.'}, s[i]) {
			return "", "", fmt.Errorf("invalid unquoted char (%v) in value, input: %s", s[i], s)
		}
	}
	return verifyReturn(s[:i], s[i:])
}

// parseLabels parses key="value" pairs from the input string
func parseLabels(input string) ([][2]string, error) {
	var labels [][2]string
	key := ""
	s := strings.TrimSpace(input)
	for len(s) != 0 {
		if key == "" {
			k, rem, err := readLabelKey(s)
			if err != nil {
				return nil, err
			}
			key = k
			s = rem
		} else {
			v, rem, err := readLabelValue(s)
			if err != nil {
				return nil, err
			}
			labels = append(labels, [2]string{key, v})
			key = ""
			s = rem
		}
	}
	if key != "" {
		labels = append(labels, [2]string{key, ""})
	}

	return labels, nil
}

func parseLine(line string) (metricName, value string, labels [][2]string, err error) {
	defer func() {
		switch metricName {
		case "success", "total", "latency":
			err = fmt.Errorf("metric name (%s) in the payload conflicts with standard metrics: (success,total,latency), ignoring", metricName)
			return
		}
	}()

	line = strings.TrimSpace(line)
	ob := strings.Index(line, "{")

	// If "{" was not found or was the last element, assume label-less metric.
	if ob == -1 || ob == len(line)-1 {
		varKV := strings.SplitN(line, " ", 2)
		if len(varKV) < 2 {
			err = fmt.Errorf("wrong var key-value format: %s", line)
			return
		}

		metricName, value = varKV[0], strings.TrimSpace(varKV[1])
		return
	}

	metricName, line = line[:ob], line[ob+1:]

	eb := strings.Index(line, "}")
	// If "}" was not found or was the last element, invalid line.
	if eb == -1 || eb == len(line)-1 {
		err = fmt.Errorf("invalid line (%s), only opening brace found", line)
		return
	}

	// Capture label string and move line-beginning forward.
	labels, err = parseLabels(line[:eb])
	if err != nil {
		return
	}
	value = strings.TrimSpace(line[eb+1:])
	return
}

// processLineDistValue processes a distribution value. It works with distribution
// values in 2 formats:
// a) a full distribution string, capturing all the details, e.g.
//
//	"dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12"
//
// a) a comma-separated list of floats, where distribution details have been
//
//	provided at the time of config, e.g.
//	"12,13,10.1,9.875,11.1"
func processLineDistValue(mVal *metrics.Distribution, val string) error {
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

func metricKey(name, target string, labels [][2]string) string {
	var parts []string
	parts = append(parts, name)
	for _, l := range append(labels, [2]string{"dst", target}) {
		parts = append(parts, fmt.Sprintf("%s=%s", l[0], l[1]))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (p *Parser) processLine(line, targetKey string) *metrics.EventMetrics {
	metricName, valueStr, labels, err := parseLine(line)
	if err != nil {
		p.l.Warningf("error while parsing line (%s): %v", line, err)
		return nil
	}

	var value metrics.Value
	// If it's a pre-configured, distribution metric.
	if dv, ok := p.distMetrics[metricName]; ok {
		d := dv.Clone().(*metrics.Distribution)
		if err := processLineDistValue(d, valueStr); err != nil {
			p.l.Warning(err.Error())
			return nil
		}
		value = d
	} else {
		value, err = metrics.ParseValueFromString(valueStr)
		if err != nil {
			p.l.Warning(err.Error())
			return nil
		}
	}

	// If not aggregating, create a new EM, add the metric and return.
	if !p.aggregate {
		return p.newEM(labels).AddMetric(metricName, value)
	} else {
		// If aggregating, create a key, find if we already have an EM with that key
		// if yes, update that metric, or create a new metric.
		key := metricKey(metricName, targetKey, labels)

		p.mu.Lock()
		defer p.mu.Unlock()

		if em := p.aggregatedMetrics[key]; em != nil {
			if err := em.Metric(metricName).Add(value); err != nil {
				p.l.Warningf("error updating metric %s with val %s: %v", metricName, value, err)
				return nil
			}
			return em.Clone()
		}

		em := p.newEM(labels).AddMetric(metricName, value)
		p.aggregatedMetrics[key] = em
		return em.Clone()
	}
}

// payloadLineMetrics parses a payload lines as metrics, and for each line in
// correct format, either updates an existing EM, or create a new one.
func (p *Parser) lineBasedMetrics(text []byte, targetKey string) []*metrics.EventMetrics {
	var results []*metrics.EventMetrics
	for _, line := range strings.Split(string(text), "\n") {
		if p.lineAcceptRe != nil && !p.lineAcceptRe.MatchString(line) {
			continue
		}
		if p.lineRejectRe != nil && p.lineRejectRe.MatchString(line) {
			continue
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		em := p.processLine(line, targetKey)
		if em == nil {
			continue
		}
		results = append(results, em)
	}
	return results
}
