// Copyright 2017-2021 The Cloudprober Authors.
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
Package prometheus provides a prometheus surfacer for Cloudprober. Prometheus
surfacer exports incoming metrics over a web interface in a format that
prometheus understands (http://prometheus.io).

This surfacer processes each incoming EventMetrics and holds the latest value
and timestamp for each metric in memory. These metrics are made available
through a web URL (default: /metrics), which Prometheus scrapes at a regular
interval.

Example /metrics page:
# TYPE sent counter
sent{ptype="dns",probe="vm-to-public-dns",dst="8.8.8.8"} 181299 1497330037000
sent{ptype="ping",probe="vm-to-public-dns",dst="8.8.4.4"} 362600 1497330037000
# TYPE rcvd counter
rcvd{ptype="dns",probe="vm-to-public-dns",dst="8.8.8.8"} 181234 1497330037000
rcvd{ptype="ping",probe="vm-to-public-dns",dst="8.8.4.4"} 362600 1497330037000
*/
package prometheus

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers/internal/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto"
)

var (
	metricsPrefixFlag    = flag.String("prometheus_metrics_prefix", "", "Metrics prefix")
	includeTimestampFlag = flag.Bool("prometheus_include_timestamp", false, "Include timestamp in metrics")
)

// Prometheus metric and label names should match the following regular
// expressions. Since, "-" is commonly used in metric and label names, we
// replace it by "_". If a name still doesn't match the regular expression, we
// ignore it with a warning log message.
const (
	ValidMetricNameRegex = "^[a-zA-Z_:]([a-zA-Z0-9_:])*$"
	ValidLabelNameRegex  = "^[a-zA-Z_]([a-zA-Z0-9_])*$"
)

const histogram = "histogram"

// queriesQueueSize defines how many queries can we queue before we start
// blocking on previous queries to finish.
const queriesQueueSize = 10

// Prometheus does not take metrics that have passed 10 minutes. We delete
// metrics that are older than this time.
const metricExpirationTime = 10 * time.Minute

var (
	// Cache of EventMetric label to prometheus label mapping. We use it to
	// quickly lookup if we have already seen a label and we have a prometheus
	// label corresponding to it.
	promLabelNames = make(map[string]string)

	// Cache of EventMetric metric to prometheus metric mapping. We use it to
	// quickly lookup if we have already seen a metric and we have a prometheus
	// metric name corresponding to it.
	promMetricNames = make(map[string]string)
)

type promMetric struct {
	typ      string
	data     map[string]*dataPoint
	dataKeys []string // To keep data keys ordered
}

type dataPoint struct {
	value     string
	timestamp int64
}

// httpWriter is a wrapper for http.ResponseWriter that includes a channel
// to signal the completion of the writing of the response.
type httpWriter struct {
	w        http.ResponseWriter
	doneChan chan struct{}
}

// defaultBoolEnum is used to record if a boolean flag (or config option) was
// set explicitly or not.
type defaultBoolEnum int8

const (
	defaultBehavior defaultBoolEnum = 0 // not set explicitly
	explicitTrue    defaultBoolEnum = 1
	explicitFalse   defaultBoolEnum = 2
)

var defaultBoolMap = map[bool]defaultBoolEnum{
	false: explicitFalse,
	true:  explicitTrue,
}

func includeTimestamp(c *configpb.SurfacerConf) defaultBoolEnum {
	// Config option has highest priority if set explicitly.
	if c.IncludeTimestamp != nil {
		return defaultBoolMap[c.GetIncludeTimestamp()]
	}

	// Next check if flag was set explicitly.
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "prometheus_include_timestamp" {
			found = true
			return
		}
	})
	if found {
		return defaultBoolMap[*includeTimestampFlag]
	}

	return defaultBehavior
}

// PromSurfacer implements a prometheus surfacer for Cloudprober. PromSurfacer
// organizes metrics into a two-level data structure:
//  1. Metric name -> PromMetric data structure dict.
//  2. A PromMetric organizes data associated with a metric in a
//     Data key -> Data point map, where data point consists of a value
//     and timestamp.
//
// Data key represents a unique combination of metric name and labels.
type PromSurfacer struct {
	c                        *configpb.SurfacerConf // Configuration
	opts                     *options.Options
	includeTimestamp         defaultBoolEnum
	disableMetricsExpiration defaultBoolEnum
	prefix                   string                     // Metrics prefix, e.g. "cloudprober_"
	emChan                   chan *metrics.EventMetrics // Buffered channel to store incoming EventMetrics
	metrics                  map[string]*promMetric     // Metric name to promMetric mapping
	metricNames              []string                   // Metric names, to keep names ordered.
	queryChan                chan *httpWriter           // Query channel
	l                        *logger.Logger

	// A handler that takes a promMetric and a dataKey and writes the
	// corresponding metric string to the provided io.Writer.
	dataWriter func(w io.Writer, pm *promMetric, dataKey string)

	// Regexes for metric and label names.
	metricNameRe *regexp.Regexp
	labelNameRe  *regexp.Regexp
}

// New returns a prometheus surfacer based on the config provided. It sets up a
// goroutine to process both the incoming EventMetrics and the web requests for
// the URL handler /metrics.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*PromSurfacer, error) {
	if config == nil {
		config = &configpb.SurfacerConf{}
	}
	ps := &PromSurfacer{
		c:            config,
		opts:         opts,
		emChan:       make(chan *metrics.EventMetrics, config.GetMetricsBufferSize()),
		queryChan:    make(chan *httpWriter, queriesQueueSize),
		metrics:      make(map[string]*promMetric),
		metricNameRe: regexp.MustCompile(ValidMetricNameRegex),
		labelNameRe:  regexp.MustCompile(ValidLabelNameRegex),
		prefix:       *metricsPrefixFlag,
		l:            l,
	}

	ps.includeTimestamp = includeTimestamp(ps.c)

	if ps.c.MetricsPrefix != nil {
		ps.prefix = ps.c.GetMetricsPrefix()
	}

	switch ps.includeTimestamp {
	case explicitTrue:
		ps.dataWriter = func(w io.Writer, pm *promMetric, k string) {
			fmt.Fprintf(w, "%s %s %d\n", k, pm.data[k].value, pm.data[k].timestamp)
		}
	case explicitFalse:
		ps.dataWriter = func(w io.Writer, pm *promMetric, k string) {
			fmt.Fprintf(w, "%s %s\n", k, pm.data[k].value)
		}
	default:
		ps.dataWriter = func(w io.Writer, pm *promMetric, k string) {
			if pm.typ == "gauge" {
				fmt.Fprintf(w, "%s %s %d\n", k, pm.data[k].value, pm.data[k].timestamp)
				return
			}
			fmt.Fprintf(w, "%s %s\n", k, pm.data[k].value)
		}
	}

	ps.disableMetricsExpiration = ps.disableMetricsExpirationF()

	// Start a goroutine to process the incoming EventMetrics as well as
	// the incoming web queries. To avoid data access race conditions, we do
	// one thing at a time.
	go func() {
		staleMetricDeleteTimer := time.NewTicker(metricExpirationTime)
		defer staleMetricDeleteTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				ps.l.Infof("Context canceled, stopping the input/output processing loop.")
				return
			case em := <-ps.emChan:
				ps.record(em)
			case hw := <-ps.queryChan:
				ps.writeData(hw.w)
				close(hw.doneChan)
			case <-staleMetricDeleteTimer.C:
				ps.deleteExpiredMetrics()
			}
		}
	}()

	err := state.AddWebHandler(ps.c.GetMetricsUrl(), func(w http.ResponseWriter, r *http.Request) {
		// doneChan is used to track the completion of the response writing. This is
		// required as response is written in a different goroutine.
		doneChan := make(chan struct{}, 1)
		ps.queryChan <- &httpWriter{w, doneChan}
		<-doneChan
	})
	if err != nil {
		return nil, err
	}

	l.Infof("Initialized prometheus exporter at the URL: %s", ps.c.GetMetricsUrl())
	return ps, nil
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually processes the data and updates the in-memory
// database.
func (ps *PromSurfacer) Write(_ context.Context, em *metrics.EventMetrics) {
	select {
	case ps.emChan <- em:
	default:
		ps.l.Errorf("PromSurfacer's write channel is full, dropping new data.")
	}
}

func (ps *PromSurfacer) disableMetricsExpirationF() defaultBoolEnum {
	if ps.c != nil && ps.c.DisableMetricsExpiration != nil {
		return defaultBoolMap[*ps.c.DisableMetricsExpiration]
	}

	// If we are never including timestamp, we disable metrics expiration
	// and vice versa.
	return map[defaultBoolEnum]defaultBoolEnum{
		explicitFalse:   explicitTrue,
		explicitTrue:    explicitFalse,
		defaultBehavior: defaultBehavior,
	}[ps.includeTimestamp]
}

func promType(em *metrics.EventMetrics) string {
	switch em.Kind {
	case metrics.CUMULATIVE:
		return "counter"
	case metrics.GAUGE:
		return "gauge"
	default:
		return "unknown"
	}
}

// promTime converts time.Time to Unix milliseconds.
func promTime(t time.Time) int64 {
	return t.UnixNano() / (1000 * 1000)
}

func (ps *PromSurfacer) recordMetric(metricName, key, value string, em *metrics.EventMetrics, typ string) {
	// Recognized metric
	if pm := ps.metrics[metricName]; pm != nil {
		// Recognized metric name and labels combination.
		if pm.data[key] != nil {
			pm.data[key].value = value
			pm.data[key].timestamp = promTime(em.Timestamp)
			return
		}
		pm.data[key] = &dataPoint{
			value:     value,
			timestamp: promTime(em.Timestamp),
		}
		pm.dataKeys = append(pm.dataKeys, key)
	} else {
		// Newly discovered metric name.
		if typ == "" {
			typ = promType(em)
		}
		ps.metrics[metricName] = &promMetric{
			typ: typ,
			data: map[string]*dataPoint{
				key: {
					value:     value,
					timestamp: promTime(em.Timestamp),
				},
			},
			dataKeys: []string{key},
		}
		ps.metricNames = append(ps.metricNames, metricName)
	}
}

// checkLabelName finds a prometheus label name for an incoming label. If label
// is found to be invalid even after some basic conversions, a zero string is
// returned.
func (ps *PromSurfacer) checkLabelName(k string) string {
	// Before checking with regex, see if this label name is
	// already known. This block will be entered only once per
	// label name.
	if promLabel, ok := promLabelNames[k]; ok {
		return promLabel
	}

	// We'll come here only once per label name.
	ps.l.Debugf("Checking validity of new label: %s", k)

	// Prometheus doesn't support "-" in metric names.
	labelName := strings.Replace(k, "-", "_", -1)
	if !ps.labelNameRe.MatchString(labelName) {
		// Explicitly store a zero string so that we don't check it again.
		promLabelNames[k] = ""
		ps.l.Warningf("Ignoring invalid prometheus label name: %s", k)
		return ""
	}
	promLabelNames[k] = labelName
	return labelName
}

// promMetricName finds a prometheus metric name for an incoming metric. If metric
// is found to be invalid even after some basic conversions, a zero string is
// returned.
func (ps *PromSurfacer) promMetricName(k string) string {
	k = ps.prefix + k

	// Before checking with regex, see if this metric name is
	// already known. This block will be entered only once per
	// metric name.
	if metricName, ok := promMetricNames[k]; ok {
		return metricName
	}

	// We'll come here only once per metric name.
	ps.l.Debugf("Checking validity of new metric: %s", k)

	// Prometheus doesn't support "-" in metric names.
	metricName := strings.Replace(k, "-", "_", -1)
	if !ps.metricNameRe.MatchString(metricName) {
		// Explicitly store a zero string so that we don't check it again.
		promMetricNames[k] = ""
		ps.l.Warningf("Ignoring invalid prometheus metric name: %s", k)
		return ""
	}
	promMetricNames[k] = metricName
	return metricName
}

func dataKey(metricName string, labels []string) string {
	return metricName + "{" + strings.Join(labels, ",") + "}"
}

func recordMap[T int64 | float64](ps *PromSurfacer, m *metrics.Map[T], em *metrics.EventMetrics, pMetricName string, labels []string) {
	labelName := ps.checkLabelName(m.MapName)
	if labelName == "" {
		return
	}
	for _, k := range m.Keys() {
		key := dataKey(pMetricName, append(labels, labelName+"=\""+k+"\""))
		ps.recordMetric(pMetricName, key, metrics.MapValueToString(m.GetKey(k)), em, "")
	}
}

// record processes the incoming EventMetrics and updates the in-memory
// database.
//
// Since prometheus doesn't support certain metrics.Value types, we handle them
// differently.
//
// metrics.Map value type:  We break Map values into multiple data keys, with
// each map key corresponding to a label in the data key.
// For example, "resp-code map:code 200:45 500:2" gets converted into:
//
//	resp-code{code=200} 45
//	resp-code{code=500}  2
//
// metrics.String value type: We convert string value type into a data key with
// val="value" label.
// For example, "version cloudprober-20170608-RC00" gets converted into:
//
//	version{val=cloudprober-20170608-RC00} 1
func (ps *PromSurfacer) record(em *metrics.EventMetrics) {
	var labels []string
	for _, k := range em.LabelsKeys() {
		if labelName := ps.checkLabelName(k); labelName != "" {
			labels = append(labels, labelName+"=\""+em.Label(k)+"\"")
		}
	}

	for _, metricName := range em.MetricsKeys() {
		if !ps.opts.AllowMetric(metricName) {
			continue
		}
		pMetricName := ps.promMetricName(metricName)
		if pMetricName == "" {
			// No prometheus metric name found for this metric.
			continue
		}
		val := em.Metric(metricName)

		switch v := val.(type) {
		case *metrics.Map[int64]:
			recordMap(ps, v, em, pMetricName, labels)
		case *metrics.Map[float64]:
			recordMap(ps, v, em, pMetricName, labels)
		// Distribution values get expanded into metrics with extra label "le".
		case *metrics.Distribution:
			d := v.Data()
			var val int64
			ps.recordMetric(pMetricName, dataKey(pMetricName+"_sum", labels), strconv.FormatFloat(d.Sum, 'f', -1, 64), em, histogram)
			ps.recordMetric(pMetricName, dataKey(pMetricName+"_count", labels), strconv.FormatInt(d.Count, 10), em, histogram)
			for i := range d.LowerBounds {
				val += d.BucketCounts[i]
				var lb string
				if i == len(d.LowerBounds)-1 {
					lb = "+Inf"
				} else {
					lb = strconv.FormatFloat(d.LowerBounds[i+1], 'f', -1, 64)
				}
				labelsWithBucket := append(labels, "le=\""+lb+"\"")
				ps.recordMetric(pMetricName, dataKey(pMetricName+"_bucket", labelsWithBucket), strconv.FormatInt(val, 10), em, histogram)
			}
		case metrics.String:
			newLabels := append(labels, "val="+val.String())
			ps.recordMetric(pMetricName, dataKey(pMetricName, newLabels), "1", em, "")

		// All other value types, mostly numerical types.
		default:
			ps.recordMetric(pMetricName, dataKey(pMetricName, labels), val.String(), em, "")
		}
	}
}

// writeData writes metrics data on w io.Writer
func (ps *PromSurfacer) writeData(w io.Writer) {
	for _, name := range ps.metricNames {
		pm := ps.metrics[name]
		fmt.Fprintf(w, "# TYPE %s %s\n", name, pm.typ)
		for _, k := range pm.dataKeys {
			ps.dataWriter(w, pm, k)
		}
	}
}

// deleteExpiredMetrics clears the metric expired in PromSurfacer.
// Note from manugarg: We can possibly optimize this by recording expired
// keys while serving the metrics, and deleting them based on the timer.
func (ps *PromSurfacer) deleteExpiredMetrics() {
	if ps.disableMetricsExpiration == explicitTrue {
		return
	}

	staleTimeThreshold := promTime(time.Now()) - metricExpirationTime.Milliseconds()

	for _, name := range ps.metricNames {
		pm := ps.metrics[name]

		// For default behavior, we don't expire counter metrics.
		if ps.disableMetricsExpiration == defaultBehavior && pm.typ == "counter" {
			continue
		}

		var expiredMetricsKeys []string
		for metricKey, v := range pm.data {
			if v.timestamp < staleTimeThreshold {
				expiredMetricsKeys = append(expiredMetricsKeys, metricKey)
			}
		}

		for _, expiredMetricKey := range expiredMetricsKeys {
			delete(pm.data, expiredMetricKey)
			pm.dataKeys = deleteFromSlice(pm.dataKeys, expiredMetricKey)
		}
	}
}

// deleteFromSlice delete target on slice
func deleteFromSlice(stringSlice []string, targetData string) []string {
	targetIndex := -1
	for i, data := range stringSlice {
		if data == targetData {
			targetIndex = i
			break
		}
	}

	if targetIndex != -1 {
		stringSlice = append(stringSlice[:targetIndex], stringSlice[targetIndex+1:]...)
	}
	return stringSlice
}
