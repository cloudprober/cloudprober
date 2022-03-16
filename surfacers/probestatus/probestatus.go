// Copyright 2022 The Cloudprober Authors.
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
Package probestatus implements a surfacer that exposes probes' status over web
interface. This surfacer builds an in-memory timeseries database from the
incoming EventMetrics.

Example /probestatus page:
*/
package probestatus

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/probestatus/proto"
)

const (
	metricsBufferSize = 10000
)

const histogram = "histogram"

// queriesQueueSize defines how many queries can we queue before we start
// blocking on previous queries to finish.
const queriesQueueSize = 10

// httpWriter is a wrapper for http.ResponseWriter that includes a channel
// to signal the completion of the writing of the response.
type httpWriter struct {
	w        http.ResponseWriter
	doneChan chan struct{}
}

type timeseries struct {
	a              []*datum
	latest, oldest int
	res            time.Duration
	currentTS      time.Time
}

func newTimeseries(resolution time.Duration, maxPoints int) *timeseries {
	if resolution == 0 {
		resolution = time.Minute
	}
	return &timeseries{
		a:   make([]*datum, maxPoints),
		res: resolution,
	}
}

func (ts *timeseries) addDatum(t time.Time, d *datum) {
	tt := t.Truncate(ts.res)
	// Need a new bucket
	if tt.After(ts.currentTS) {
		// Move
		ts.latest = (ts.latest + 1) % len(ts.a)
		if ts.latest == ts.oldest {
			ts.oldest++
		}
	}
	// Same bucket but newer data
	if t.After(ts.currentTS) {
		ts.currentTS = tt
		ts.a[ts.latest] = d
		return
	}
}

func (ts *timeseries) getRecentData(td time.Duration) []*datum {
	size := int(td / ts.res)
	if size > len(ts.a) || size == 0 {
		size = len(ts.a)
	}

	oldestIndex := ts.latest - (size - 1)
	if oldestIndex >= 0 {
		if oldestIndex == 0 {
			oldestIndex = 1
		}
		return append([]*datum{}, ts.a[oldestIndex:ts.latest+1]...)
	}

	if ts.oldest == 0 {
		return append([]*datum{}, ts.a[1:ts.latest+1]...)
	}

	otherSide := len(ts.a) + oldestIndex
	return append([]*datum{}, append(ts.a[otherSide:], ts.a[:ts.latest+1]...)...)
}

type datum struct {
	success, total int64
	latency        metrics.Value
}

// Surfacer implements a status surfacer for Cloudprober.
type Surfacer struct {
	c         *configpb.SurfacerConf // Configuration
	opts      *options.Options
	emChan    chan *metrics.EventMetrics // Buffered channel to store incoming EventMetrics
	queryChan chan *httpWriter           // Query channel
	l         *logger.Logger

	resolution   time.Duration
	metrics      map[string]map[string]*timeseries
	probeNames   []string
	probeTargets map[string][]string
}

// New returns a probestatus surfacer based on the config provided. It sets up
// a goroutine to process both the incoming EventMetrics and the web requests
// for the URL handler /metrics.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) *Surfacer {
	if config == nil {
		config = &configpb.SurfacerConf{}
	}

	res := time.Duration(config.GetResolutionSec()) * time.Second
	if res == 0 {
		res = time.Minute
	}

	ps := &Surfacer{
		c:            config,
		opts:         opts,
		emChan:       make(chan *metrics.EventMetrics, metricsBufferSize),
		queryChan:    make(chan *httpWriter, queriesQueueSize),
		metrics:      make(map[string]map[string]*timeseries),
		probeTargets: make(map[string][]string),
		resolution:   res,
		l:            l,
	}

	// Start a goroutine to process the incoming EventMetrics as well as
	// the incoming web queries. To avoid data access race conditions, we do
	// one thing at a time.
	go func() {
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
			}
		}
	}()

	http.HandleFunc(config.GetUrl(), func(w http.ResponseWriter, r *http.Request) {
		// doneChan is used to track the completion of the response writing. This is
		// required as response is written in a different goroutine.
		doneChan := make(chan struct{}, 1)
		ps.queryChan <- &httpWriter{w, doneChan}
		<-doneChan
	})

	l.Infof("Initialized status surfacer at the URL: %s", "probesstatus")
	return ps
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually processes the data and updates the in-memory
// database.
func (ps *Surfacer) Write(_ context.Context, em *metrics.EventMetrics) {
	select {
	case ps.emChan <- em:
	default:
		ps.l.Errorf("Surfacer's write channel is full, dropping new data.")
	}
}

// record processes the incoming EventMetrics and updates the in-memory
// database.
func (ps *Surfacer) record(em *metrics.EventMetrics) {
	probeName, targetName := em.Label("probe"), em.Label("dst")
	if probeName == "sysvars" || em.Metric("total") == nil {
		return
	}

	total, totalOk := em.Metric("total").(metrics.NumValue)
	success, successOk := em.Metric("success").(metrics.NumValue)
	if !totalOk || !successOk {
		return
	}

	probeTS := ps.metrics[probeName]
	if probeTS == nil {
		probeTS = make(map[string]*timeseries)
		ps.metrics[probeName] = probeTS
		ps.probeNames = append(ps.probeNames, probeName)
	}

	targetTS := probeTS[targetName]
	if targetTS == nil {
		targetTS = newTimeseries(ps.resolution, int(ps.c.GetMaxPoints()))
		probeTS[targetName] = targetTS
		ps.probeTargets[probeName] = append(ps.probeTargets[probeName], targetName)
	}

	targetTS.addDatum(em.Timestamp, &datum{
		total:   total.Int64(),
		success: success.Int64(),
		latency: em.Metric("latency").Clone(),
	})
}

func (ps *Surfacer) computeDelta(durations []time.Duration, data []*datum) (totalDelta, successDelta []int64) {
	lastData := data[len(data)-1]
	for _, td := range durations {
		numPoints := int(td / ps.resolution)
		start := len(data) - 1 - numPoints
		if start < 0 {
			start = 0
		}

		totalDelta = append(totalDelta, lastData.total-data[start].total)
		successDelta = append(successDelta, lastData.success-data[start].success)
	}
	return
}

func (ps *Surfacer) probeStatus(probeName string, durations []time.Duration) ([]string, []string) {
	var lines, debugLines []string

	for _, targetName := range ps.probeTargets[probeName] {
		lines = append(lines, "<tr><td><b>"+targetName+"</b></td>")
		ts := ps.metrics[probeName][targetName]
		data := ts.getRecentData(24 * time.Hour)

		totalDelta, successDelta := ps.computeDelta(durations, data)
		for i := range durations {
			lines = append(lines, fmt.Sprintf("<td>%.3f</td>", float64(successDelta[i])/float64(totalDelta[i])))
		}

		debugLines = append(debugLines, fmt.Sprintf("<b>Target: %s, Oldest timestamp: %s</b><br>",
			targetName, ts.currentTS.Add(time.Duration(-len(data))*ts.res)))

		for _, i := range []int{0, len(data) - 1} {
			d := data[i]
			debugLines = append(debugLines, fmt.Sprintf("#%d total=%d, success=%d, latency=%s <br/>", i, d.total, d.success, d.latency.String()))
		}
	}
	return lines, debugLines
}

func (ps *Surfacer) writeData(w io.Writer) {
	durations := []time.Duration{5 * time.Minute, 30 * time.Minute, 1 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour}
	probesStatus := make(map[string]template.HTML)
	probesStatusDebug := make(map[string]template.HTML)

	for _, probeName := range ps.probeNames {
		probeLines, probeDebugLines := ps.probeStatus(probeName, durations)
		probesStatus[probeName] = template.HTML(strings.Join(probeLines, "\n"))
		probesStatusDebug[probeName] = template.HTML(strings.Join(probeDebugLines, "\n"))
	}

	var statusBuf bytes.Buffer

	tmpl, err := template.New("statusTmpl").Parse(probeStatusTmpl)
	if err != nil {
		ps.l.Errorf("Error parsing probe status template: %v", err)
		return
	}
	tmpl.Execute(&statusBuf, struct {
		Durations         []time.Duration
		ProbeNames        []string
		ProbesStatus      map[string]template.HTML
		ProbesStatusDebug map[string]template.HTML
	}{
		Durations:         durations,
		ProbeNames:        ps.probeNames,
		ProbesStatus:      probesStatus,
		ProbesStatusDebug: probesStatusDebug,
	})
	fmt.Fprintf(w, statusBuf.String())
}
