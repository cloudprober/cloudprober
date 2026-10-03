// Copyright 2017-2025 The Cloudprober Authors.
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
	"maps"
	"math"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/prometheus/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers/options"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
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

// Content type of the protobuf format. Prometheus needs it to scrape native
// histograms.
const protobufContentType = "application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=delimited"

// queriesQueueSize defines how many queries can we queue before we start
// blocking on previous queries to finish.
const queriesQueueSize = 10

// Prometheus generates warnings while scraping samples whose timestamp is more
// than 10 minutes old. We delete timestamped metrics once they get that stale.
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

	// Labels as `name="value"` strings. labels are the EventMetrics labels;
	// all the data points from an EventMetrics share them. extraLabel is the
	// label that's specific to this data point, if any: map key for maps, and
	// value for strings. Text format gets the labels from the data key; these
	// are for the protobuf format.
	labels     []string
	extraLabel string

	// Set only for distributions (histograms). We keep the distribution as is,
	// and expand it into _sum, _count and _bucket series while writing.
	dist *metrics.DistributionData
}

// httpWriter is a wrapper for http.ResponseWriter that includes a channel
// to signal the completion of the writing of the response.
type httpWriter struct {
	w        http.ResponseWriter
	protobuf bool // Write metrics in the protobuf format.
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

func shouldIncludeTimestamp(c *configpb.SurfacerConf) defaultBoolEnum {
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
//     (or a distribution, for histograms) and timestamp.
//
// Data key represents a unique combination of metric name and labels.
type PromSurfacer struct {
	c                      *configpb.SurfacerConf // Configuration
	opts                   *options.Options
	includeTimestamp       defaultBoolEnum
	staleMetricsExpiration time.Duration              // For metrics without a timestamp; 0 disables.
	prefix                 string                     // Metrics prefix, e.g. "cloudprober_"
	emChan                 chan *metrics.EventMetrics // Buffered channel to store incoming EventMetrics
	metrics                map[string]*promMetric     // Metric name to promMetric mapping
	metricNames            []string                   // Metric names, to keep names ordered.
	queryChan              chan *httpWriter           // Query channel
	l                      *logger.Logger

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

	ps.includeTimestamp = shouldIncludeTimestamp(ps.c)

	if ps.c.MetricsPrefix != nil {
		ps.prefix = ps.c.GetMetricsPrefix()
	}

	// Metrics that we export without a timestamp are not subject to the 10m
	// limit above, but we still drop them once they've been stale for this
	// long, so that series belonging to probes and targets that have gone away
	// don't stay in the /metrics output forever. The default (see the proto)
	// is deliberately much larger than metricExpirationTime: expiring a series
	// that a slow probe is still updating is worse -- it shows up as a gap and
	// a counter reset -- than letting a dead one linger for a few hours.
	if ps.c.GetStaleMetricsExpirationSec() < 0 {
		return nil, fmt.Errorf("prometheus surfacer: stale_metrics_expiration_sec (%d) cannot be negative; use 0 to never expire", ps.c.GetStaleMetricsExpirationSec())
	}
	ps.staleMetricsExpiration = time.Duration(ps.c.GetStaleMetricsExpirationSec()) * time.Second

	// Start a goroutine to process the incoming EventMetrics as well as
	// the incoming web queries. To avoid data access race conditions, we do
	// one thing at a time.
	// Sweep at least as often as the shortest deadline we have to honor,
	// otherwise a stale_metrics_expiration_sec below metricExpirationTime would
	// be silently rounded up to it.
	sweepInterval := metricExpirationTime
	if ps.staleMetricsExpiration > 0 && ps.staleMetricsExpiration < sweepInterval {
		sweepInterval = ps.staleMetricsExpiration
	}

	go func() {
		staleMetricDeleteTimer := time.NewTicker(sweepInterval)
		defer staleMetricDeleteTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				ps.l.Infof("Context canceled, stopping the input/output processing loop.")
				return
			case em := <-ps.emChan:
				ps.record(em)
			case hw := <-ps.queryChan:
				if hw.protobuf {
					hw.w.Header().Set("Content-Type", protobufContentType)
					ps.writeProtobuf(hw.w)
				} else {
					ps.writeData(hw.w)
				}
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
		ps.queryChan <- &httpWriter{w: w, protobuf: acceptsProtobuf(r.Header.Get("Accept")), doneChan: doneChan}
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

// isTimestamped returns whether we export metrics of the given prometheus type
// with a timestamp.
func (ps *PromSurfacer) isTimestamped(typ string) bool {
	switch ps.includeTimestamp {
	case explicitTrue:
		return true
	case explicitFalse:
		return false
	default:
		return typ == "gauge"
	}
}

// expirationAge returns how long a metric of the given prometheus type is kept
// after its last update. A zero duration means it's never expired.
func (ps *PromSurfacer) expirationAge(typ string) time.Duration {
	if ps.c.GetDisableMetricsExpiration() {
		return 0
	}

	// Metrics we export with a timestamp have to go once Prometheus would warn
	// about them; the rest are kept until they're simply stale.
	if ps.isTimestamped(typ) {
		return metricExpirationTime
	}
	return ps.staleMetricsExpiration
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

func (ps *PromSurfacer) recordMetric(metricName, key string, dp dataPoint, em *metrics.EventMetrics, typ string) {
	dp.timestamp = promTime(em.Timestamp)

	pm := ps.metrics[metricName]
	if pm == nil {
		// Newly discovered metric name.
		if typ == "" {
			typ = promType(em)
		}
		pm = &promMetric{typ: typ, data: make(map[string]*dataPoint)}
		ps.metrics[metricName] = pm
		ps.metricNames = append(ps.metricNames, metricName)
	}

	// Recognized metric name and labels combination.
	if pm.data[key] != nil {
		*pm.data[key] = dp
		return
	}
	// Store a copy, so that dp itself doesn't get allocated on the heap for
	// each update.
	newDP := dp
	pm.data[key] = &newDP
	pm.dataKeys = append(pm.dataKeys, key)
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
		mapLabel := labelName + "=\"" + k + "\""
		key := dataKey(pMetricName, append(labels, mapLabel))
		ps.recordMetric(pMetricName, key, dataPoint{value: metrics.MapValueToString(m.GetKey(k)), labels: labels, extraLabel: mapLabel}, em, "")
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
		case *metrics.Distribution:
			// We keep the distribution data until scrape time, so the
			// distribution must not change after it has been written to the
			// surfacers. Probes write a clone of their distributions.
			ps.recordMetric(pMetricName, dataKey(pMetricName, labels), dataPoint{dist: v.Data(), labels: labels}, em, histogram)
		case metrics.String:
			valLabel := "val=" + val.String()
			ps.recordMetric(pMetricName, dataKey(pMetricName, append(labels, valLabel)), dataPoint{value: "1", labels: labels, extraLabel: valLabel}, em, "")

		// All other value types, mostly numerical types.
		default:
			ps.recordMetric(pMetricName, dataKey(pMetricName, labels), dataPoint{value: val.String(), labels: labels}, em, "")
		}
	}
}

// appendLineEnd appends the timestamp, if needed, and a newline to buf.
func appendLineEnd(buf []byte, withTimestamp bool, ts int64) []byte {
	if withTimestamp {
		buf = append(buf, ' ')
		buf = strconv.AppendInt(buf, ts, 10)
	}
	return append(buf, '\n')
}

// appendHistogram appends a distribution as _sum, _count and _bucket lines to
// buf. Bucket lines have cumulative counts and an extra label, "le", for the
// bucket's upper bound.
func appendHistogram(buf []byte, name, labels string, dp *dataPoint, withTimestamp bool) []byte {
	d := dp.dist

	// Appends "<name><suffix>{<labels>", leaving the label set open so that
	// _bucket lines can add "le".
	appendKeyStart := func(suffix string) {
		buf = append(buf, name...)
		buf = append(buf, suffix...)
		buf = append(buf, '{')
		buf = append(buf, labels...)
	}

	appendKeyStart("_sum")
	buf = append(buf, "} "...)
	buf = strconv.AppendFloat(buf, d.Sum, 'f', -1, 64)
	buf = appendLineEnd(buf, withTimestamp, dp.timestamp)

	appendKeyStart("_count")
	buf = append(buf, "} "...)
	buf = strconv.AppendInt(buf, d.Count, 10)
	buf = appendLineEnd(buf, withTimestamp, dp.timestamp)

	appendBucket := func(le float64, count int64) {
		appendKeyStart("_bucket")
		if labels != "" {
			buf = append(buf, ',')
		}
		buf = append(buf, "le=\""...)
		buf = strconv.AppendFloat(buf, le, 'f', -1, 64)
		buf = append(buf, "\"} "...)
		buf = strconv.AppendInt(buf, count, 10)
		buf = appendLineEnd(buf, withTimestamp, dp.timestamp)
	}

	// Native histograms can have hundreds of buckets, and they are exported
	// fully only in the protobuf format. In the text format, we write only the
	// +Inf bucket for them.
	if d.Native != nil {
		appendBucket(math.Inf(1), d.Count)
		return buf
	}

	var count int64
	for i := range d.LowerBounds {
		count += d.BucketCounts[i]
		le := math.Inf(1)
		if i < len(d.LowerBounds)-1 {
			le = d.LowerBounds[i+1]
		}
		appendBucket(le, count)
	}
	return buf
}

// writeData writes metrics data on w io.Writer. We build the output in a
// buffer, instead of using fmt.Fprintf, to avoid allocations for each line.
func (ps *PromSurfacer) writeData(w io.Writer) {
	buf := make([]byte, 0, 64*1024)
	for _, name := range ps.metricNames {
		pm := ps.metrics[name]
		buf = append(buf, "# TYPE "...)
		buf = append(buf, name...)
		buf = append(buf, ' ')
		buf = append(buf, pm.typ...)
		buf = append(buf, '\n')

		withTimestamp := ps.isTimestamped(pm.typ)
		for _, k := range pm.dataKeys {
			dp := pm.data[k]
			if dp.dist != nil {
				// Data key is "<name>{<labels>}".
				buf = appendHistogram(buf, name, k[len(name)+1:len(k)-1], dp, withTimestamp)
			} else {
				buf = append(buf, k...)
				buf = append(buf, ' ')
				buf = append(buf, dp.value...)
				buf = appendLineEnd(buf, withTimestamp, dp.timestamp)
			}

			// Keep the buffer small.
			if len(buf) > 32*1024 {
				w.Write(buf)
				buf = buf[:0]
			}
		}
	}
	w.Write(buf)
}

// acceptsProtobuf returns true if the Accept header prefers the protobuf format
// over the text format. Prometheus asks for the protobuf format when it's
// configured to scrape native histograms.
func acceptsProtobuf(accept string) bool {
	var bestQ float64
	var protobuf bool
	for _, part := range strings.Split(accept, ",") {
		mediaType, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		q := 1.0
		if params["q"] != "" {
			if q, err = strconv.ParseFloat(params["q"], 64); err != nil {
				continue
			}
		}
		// Media types with the same q value are preferred in the given order.
		if q <= bestQ {
			continue
		}
		bestQ = q
		protobuf = mediaType == "application/vnd.google.protobuf" &&
			params["proto"] == "io.prometheus.client.MetricFamily" &&
			(params["encoding"] == "" || params["encoding"] == "delimited")
	}
	return protobuf
}

// nativeSpansAndDeltas converts native histogram buckets to the protobuf
// format: spans of consecutive buckets, and the count of each bucket as the
// delta from the previous one.
func nativeSpansAndDeltas(buckets map[int]int64) ([]*dto.BucketSpan, []int64) {
	var spans []*dto.BucketSpan
	var deltas []int64
	var prevKey int
	var prevCount int64
	for i, k := range slices.Sorted(maps.Keys(buckets)) {
		if i == 0 || k != prevKey+1 {
			// First span's offset is the bucket index, others' is the gap from
			// the previous span.
			offset := k
			if i != 0 {
				offset = k - prevKey - 1
			}
			spans = append(spans, &dto.BucketSpan{Offset: proto.Int32(int32(offset)), Length: proto.Uint32(0)})
		}
		*spans[len(spans)-1].Length++
		deltas = append(deltas, buckets[k]-prevCount)
		prevKey, prevCount = k, buckets[k]
	}
	return spans, deltas
}

// protobufLabels returns data point's labels for the protobuf format. We build
// them from the `name="value"` strings while writing, so that we don't pay
// for them if nobody scrapes the protobuf format.
func protobufLabels(dp *dataPoint) []*dto.LabelPair {
	var pairs []*dto.LabelPair
	add := func(label string) {
		name, value, _ := strings.Cut(label, "=")
		// Remove the quotes around the value.
		pairs = append(pairs, &dto.LabelPair{Name: proto.String(name), Value: proto.String(value[1 : len(value)-1])})
	}
	for _, l := range dp.labels {
		add(l)
	}
	if dp.extraLabel != "" {
		add(dp.extraLabel)
	}
	return pairs
}

func protobufHistogram(d *metrics.DistributionData) *dto.Histogram {
	h := &dto.Histogram{
		SampleCount: proto.Uint64(uint64(d.Count)),
		SampleSum:   proto.Float64(d.Sum),
	}

	if d.Native == nil {
		var count int64
		// The last (+Inf) bucket is implied.
		for i := range len(d.LowerBounds) - 1 {
			count += d.BucketCounts[i]
			h.Bucket = append(h.Bucket, &dto.Bucket{
				CumulativeCount: proto.Uint64(uint64(count)),
				UpperBound:      proto.Float64(d.LowerBounds[i+1]),
			})
		}
		return h
	}

	h.Schema = proto.Int32(d.Native.Schema)
	h.ZeroThreshold = proto.Float64(0)
	h.ZeroCount = proto.Uint64(uint64(d.Native.ZeroCount))
	h.PositiveSpan, h.PositiveDelta = nativeSpansAndDeltas(d.Native.Positive)
	h.NegativeSpan, h.NegativeDelta = nativeSpansAndDeltas(d.Native.Negative)
	// Prometheus takes a histogram with no spans, zero count and zero
	// threshold for a classic histogram. Add an empty span to mark it native,
	// the same as client_golang.
	if len(h.PositiveSpan) == 0 && len(h.NegativeSpan) == 0 && d.Native.ZeroCount == 0 {
		h.PositiveSpan = []*dto.BucketSpan{{Offset: proto.Int32(0), Length: proto.Uint32(0)}}
	}
	return h
}

var protobufMetricType = map[string]dto.MetricType{
	"counter": dto.MetricType_COUNTER,
	"gauge":   dto.MetricType_GAUGE,
	histogram: dto.MetricType_HISTOGRAM,
}

// writeProtobuf writes metrics data on w io.Writer in the protobuf format: a
// length-delimited MetricFamily message for each metric.
func (ps *PromSurfacer) writeProtobuf(w io.Writer) {
	for _, name := range ps.metricNames {
		pm := ps.metrics[name]
		typ, ok := protobufMetricType[pm.typ]
		if !ok {
			typ = dto.MetricType_UNTYPED
		}
		mf := &dto.MetricFamily{Name: proto.String(name), Type: typ.Enum()}

		withTimestamp := ps.isTimestamped(pm.typ)
		for _, k := range pm.dataKeys {
			dp := pm.data[k]
			m := &dto.Metric{Label: protobufLabels(dp)}
			if withTimestamp {
				m.TimestampMs = proto.Int64(dp.timestamp)
			}

			if dp.dist != nil {
				m.Histogram = protobufHistogram(dp.dist)
			} else {
				// Parse the value we write in the text format, so that both
				// formats report the same value.
				v, err := strconv.ParseFloat(dp.value, 64)
				if err != nil {
					ps.l.Warningf("prometheus surfacer: skipping %s, invalid value: %s", k, dp.value)
					continue
				}
				switch typ {
				case dto.MetricType_COUNTER:
					m.Counter = &dto.Counter{Value: proto.Float64(v)}
				case dto.MetricType_GAUGE:
					m.Gauge = &dto.Gauge{Value: proto.Float64(v)}
				default:
					m.Untyped = &dto.Untyped{Value: proto.Float64(v)}
				}
			}
			mf.Metric = append(mf.Metric, m)
		}

		if len(mf.Metric) == 0 {
			continue
		}
		if _, err := protodelim.MarshalTo(w, mf); err != nil {
			ps.l.Warningf("prometheus surfacer: error writing metrics: %v", err)
			return
		}
	}
}

// deleteExpiredMetrics clears the metric expired in PromSurfacer.
// Note from manugarg: We can possibly optimize this by recording expired
// keys while serving the metrics, and deleting them based on the timer.
func (ps *PromSurfacer) deleteExpiredMetrics() {
	now := promTime(time.Now())
	deleted := 0

	for _, name := range ps.metricNames {
		pm := ps.metrics[name]

		expirationAge := ps.expirationAge(pm.typ)
		if expirationAge <= 0 {
			continue
		}
		staleTimeThreshold := now - expirationAge.Milliseconds()

		expiredMetricsKeys := make(map[string]bool)
		for metricKey, v := range pm.data {
			if v.timestamp < staleTimeThreshold {
				expiredMetricsKeys[metricKey] = true
			}
		}
		if len(expiredMetricsKeys) == 0 {
			continue
		}

		for metricKey := range expiredMetricsKeys {
			delete(pm.data, metricKey)
		}

		// Filter dataKeys in one pass. Deleting keys from the slice one at a
		// time would rescan it for each key, which gets expensive when a probe
		// or a target with many data keys goes away and they all expire in the
		// same sweep.
		dataKeys := pm.dataKeys[:0]
		for _, k := range pm.dataKeys {
			if !expiredMetricsKeys[k] {
				dataKeys = append(dataKeys, k)
			}
		}
		pm.dataKeys = dataKeys

		deleted += len(expiredMetricsKeys)
	}

	if deleted > 0 {
		ps.l.Infof("prometheus surfacer: deleted %d stale metric series.", deleted)
	}
}
