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
*/
package probestatus

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/probestatus/proto"
	"github.com/cloudprober/cloudprober/sysvars"
)

//go:embed static/*
var content embed.FS

const (
	metricsBufferSize = 10000
)

// queriesQueueSize defines how many queries can we queue before we start
// blocking on previous queries to finish.
const queriesQueueSize = 10

// httpWriter is a wrapper for http.ResponseWriter that includes a channel
// to signal the completion of the writing of the response.
type httpWriter struct {
	w        http.ResponseWriter
	r        *http.Request
	doneChan chan struct{}
}

type pageCache struct {
	mu         sync.RWMutex
	content    []byte
	cachedTime time.Time
	maxAge     time.Duration
}

func (pc *pageCache) contentIfValid() ([]byte, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if time.Since(pc.cachedTime) > pc.maxAge {
		return nil, false
	}
	return pc.content, true
}

func (pc *pageCache) setContent(content []byte) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.content, pc.cachedTime = content, time.Now()
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

	// Dashboard page cache.
	pageCache *pageCache

	// Dashboard Metadata
	dashDurations     []time.Duration
	dashDurationsText []string
}

// New returns a probestatus surfacer based on the config provided. It sets up
// a goroutine to process both the incoming EventMetrics and the web requests
// for the URL handler /metrics.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) *Surfacer {
	if config == nil {
		config = &configpb.SurfacerConf{}
	}

	if config.GetDisable() {
		return nil
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

	ps.dashDurations, ps.dashDurationsText = dashboardDurations(ps.resolution * time.Duration(ps.c.GetTimeseriesSize()))
	ps.pageCache = &pageCache{
		maxAge: time.Duration(ps.c.GetCacheTimeSec()) * time.Second,
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
				ps.writeData(hw)
				close(hw.doneChan)
			}
		}
	}()

	if config.GetUrl() != "/probestatus" {
		opts.HTTPServeMux.HandleFunc("/probestatus", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, config.GetUrl(), http.StatusMovedPermanently)
		})
	}
	opts.HTTPServeMux.HandleFunc(config.GetUrl(), func(w http.ResponseWriter, r *http.Request) {
		// doneChan is used to track the completion of the response writing. This is
		// required as response is written in a different goroutine.
		doneChan := make(chan struct{}, 1)
		ps.queryChan <- &httpWriter{w, r, doneChan}
		<-doneChan
	})
	opts.HTTPServeMux.Handle(config.GetUrl()+"/static/", http.StripPrefix(config.GetUrl(), http.FileServer(http.FS(content))))

	l.Infof("Initialized status surfacer at the URL: %s", "probesstatus")
	return ps
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually processes the data and updates the in-memory
// database.
func (ps *Surfacer) Write(_ context.Context, em *metrics.EventMetrics) {
	if ps == nil {
		return
	}

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
		if len(probeTS) == int(ps.c.GetMaxTargetsPerProbe())-1 {
			ps.l.Warningf("Reached the per-probe timeseries capacity (%d) with target \"%s\". All new targets will be silently dropped.", ps.c.GetMaxTargetsPerProbe(), targetName)
		}
		if len(probeTS) >= int(ps.c.GetMaxTargetsPerProbe()) {
			return
		}
		targetTS = newTimeseries(ps.resolution, int(ps.c.GetTimeseriesSize()), ps.l)
		probeTS[targetName] = targetTS
		ps.probeTargets[probeName] = append(ps.probeTargets[probeName], targetName)
	}

	targetTS.addDatum(em.Timestamp, &datum{
		total:   total.Int64(),
		success: success.Int64(),
	})
}

func (ps *Surfacer) statusTable(probeName string, durations []time.Duration) string {
	var b strings.Builder
	for _, targetName := range ps.probeTargets[probeName] {
		b.WriteString("<tr><td><b>" + targetName + "</b></td>")
		ts := ps.metrics[probeName][targetName]

		for _, td := range durations {
			t, s := ts.computeDelta(td)
			b.WriteString(fmt.Sprintf("<td>%.4f</td>", float64(s)/float64(t)))
		}
	}
	return b.String()
}

func (ps *Surfacer) debugLines(probeName string) string {
	var b strings.Builder
	for _, targetName := range ps.probeTargets[probeName] {
		ts := ps.metrics[probeName][targetName]
		b.WriteString(fmt.Sprintf("Target: %s <br>\n", targetName))
		d := ts.a[ts.oldest]
		b.WriteString(fmt.Sprintf("Oldest: total=%d, success=%d <br>\n", d.total, d.success))
		d = ts.a[ts.latest]
		b.WriteString(fmt.Sprintf("Latest: total=%d, success=%d <br>", d.total, d.success))
	}
	return b.String()
}

func (ps *Surfacer) writeData(hw *httpWriter) {
	defer func() {
		if r := recover(); r != nil {
			msg := "Unknown error"
			switch t := r.(type) {
			case string:
				msg = t
			case error:
				msg = t.Error()
			}
			http.Error(hw.w, msg, http.StatusInternalServerError)
		}
	}()

	content, valid := ps.pageCache.contentIfValid()
	if valid {
		hw.w.Write(content)
		return
	}

	startTime := sysvars.StartTime().Truncate(time.Millisecond)
	uptime := time.Since(startTime).Truncate(time.Millisecond)

	statusTable := make(map[string]template.HTML)
	debugData := make(map[string]template.HTML)
	graphData := make(map[string]template.JS)

	probes := ps.probeNames
	if v := hw.r.URL.Query()["probe"]; v != nil {
		probes = strings.Split(v[len(v)-1], ",")
	}
	maxDuration := time.Duration(ps.c.GetTimeseriesSize()) * ps.resolution
	graphOpts := graphOptsFromURL(hw.r.URL.Query(), maxDuration, ps.l)

	for _, probeName := range probes {
		statusTable[probeName] = template.HTML(ps.statusTable(probeName, ps.dashDurations))
		debugData[probeName] = template.HTML(ps.debugLines(probeName))

		// Compute graph data and convert it into JSON for embedding.
		gd := computeGraphData(ps.metrics[probeName], graphOpts)
		graphData[probeName] = template.JS(gd.JSONBytes(ps.l))
	}

	var statusBuf bytes.Buffer

	tmpl, err := template.New("statusTmpl").Parse(probeStatusTmpl)
	if err != nil {
		ps.l.Errorf("Error parsing probe status template: %v", err)
		return
	}
	err = tmpl.Execute(&statusBuf, struct {
		BaseURL           string
		Durations         []string
		ProbeNames        []string
		StatusTable       map[string]template.HTML
		GraphData         map[string]template.JS
		DebugData         map[string]template.HTML
		Version           string
		StartTime, Uptime fmt.Stringer
	}{
		BaseURL:     ps.c.GetUrl(),
		Durations:   ps.dashDurationsText,
		ProbeNames:  probes,
		StatusTable: statusTable,
		GraphData:   graphData,
		DebugData:   debugData,
		Version:     runconfig.Version(),
		StartTime:   startTime,
		Uptime:      uptime,
	})
	if err != nil {
		ps.l.Errorf("Error executing probe status template: %v", err)
		return
	}
	ps.pageCache.setContent(statusBuf.Bytes())
	hw.w.Write(statusBuf.Bytes())
}
