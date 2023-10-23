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

	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/internal/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/internal/probestatus/proto"
	"github.com/cloudprober/cloudprober/web/resources"
	"github.com/cloudprober/cloudprober/web/webutils"
)

//go:embed static/*
var content embed.FS

var statusTmpl = template.Must(template.New("statusTmpl").Parse(htmlTmpl))

const (
	metricsBufferSize = 10000
)

var dropAfterNoDataFor = 6 * time.Hour

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
	content    map[string][]byte
	cachedTime map[string]time.Time
	maxAge     time.Duration
}

func newPageCache(cacheTimeSec int) *pageCache {
	return &pageCache{
		content:    make(map[string][]byte),
		cachedTime: make(map[string]time.Time),
		maxAge:     time.Duration(cacheTimeSec) * time.Second,
	}
}

func (pc *pageCache) contentIfValid(url string) ([]byte, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if time.Since(pc.cachedTime[url]) > pc.maxAge {
		return nil, false
	}
	return pc.content[url], true
}

func (pc *pageCache) setContent(url string, content []byte) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.content[url], pc.cachedTime[url] = content, time.Now()
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
	startTime         time.Time
}

// New returns a probestatus surfacer based on the config provided. It sets up
// a goroutine to process both the incoming EventMetrics and the web requests
// for the URL handler /metrics.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*Surfacer, error) {
	if config == nil {
		config = &configpb.SurfacerConf{}
	}

	if config.GetDisable() {
		return nil, nil
	}

	if webutils.IsHandled(opts.HTTPServeMux, config.GetUrl()) {
		return nil, fmt.Errorf("probestatus surfacer URL (%s) is already registered", config.GetUrl())
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
		startTime:    sysvars.StartTime().Truncate(time.Millisecond),

		resolution: res,
		l:          l,
	}

	ps.dashDurations, ps.dashDurationsText = dashboardDurations(ps.resolution * time.Duration(ps.c.GetTimeseriesSize()))
	ps.pageCache = newPageCache(int(ps.c.GetCacheTimeSec()))

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

	opts.HTTPServeMux.HandleFunc(config.GetUrl(), func(w http.ResponseWriter, r *http.Request) {
		// doneChan is used to track the completion of the response writing. This is
		// required as response is written in a different goroutine.
		doneChan := make(chan struct{}, 1)
		ps.queryChan <- &httpWriter{w, r, doneChan}
		<-doneChan
	})

	if !webutils.IsHandled(opts.HTTPServeMux, "/probestatus") {
		opts.HTTPServeMux.Handle("/probestatus", http.RedirectHandler(config.GetUrl(), http.StatusFound))
	}

	opts.HTTPServeMux.Handle(config.GetUrl()+"/static/", http.StripPrefix(config.GetUrl(), http.FileServer(http.FS(content))))

	if !webutils.IsHandled(opts.HTTPServeMux, "/") {
		opts.HTTPServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, "/probestatus", http.StatusFound)
		})
	}

	l.Infof("Initialized status surfacer at the URL: %s", "probesstatus")
	return ps, nil
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

func (ps *Surfacer) deleteTargetWithNoLock(probeName, targetName string) {
	delete(ps.metrics[probeName], targetName)

	targets := ps.probeTargets[probeName]

	targetIndex := -1
	for i, tgt := range targets {
		if tgt == targetName {
			targetIndex = i
			break
		}
	}
	if targetIndex != -1 {
		ps.probeTargets[probeName] = append(targets[:targetIndex], targets[targetIndex+1:]...)
	}
}

func (ps *Surfacer) statusTable(probeName string) string {
	var b strings.Builder
	for _, targetName := range ps.probeTargets[probeName] {
		ts := ps.metrics[probeName][targetName]
		if ts == nil {
			continue
		}

		b.WriteString("<tr><td><b>" + targetName + "</b></td>")

		gotSomeData := false        // To track if we got some data
		var noDataFor time.Duration // No data for at least this long
		tdTmpl := "<td>%.4f</td>"   // Default table cell template

		noFurtherData := false
		maxInterval := time.Since(ts.startTime)
		for _, td := range ps.dashDurations {
			if noFurtherData {
				b.WriteString("<td align=center class=\"tooltip greyed\">...<span class=tooltiptext>No data yet</span></td>")
				continue
			}
			if td > maxInterval {
				noFurtherData = true
			}
			t, s := ts.computeDelta(td)
			if t == -1 {
				b.WriteString("<td class=greyed style=font-size:smaller>No Data</td>")
				noDataFor = td
				// After no data, ask further values stale.
				tdTmpl = "<td class=\"tooltip greyed\">%.4f<span class=tooltiptext>Stale value</span></td>"
				continue
			}
			gotSomeData = true
			b.WriteString(fmt.Sprintf(tdTmpl, float64(s)/float64(t)))
		}

		// No data for a while, drop this target.
		if noDataFor >= dropAfterNoDataFor || !gotSomeData {
			ps.deleteTargetWithNoLock(probeName, targetName)
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

	content, valid := ps.pageCache.contentIfValid(hw.r.URL.String())
	if valid {
		hw.w.Write(content)
		return
	}

	statusTable := make(map[string]template.HTML)
	debugData := make(map[string]template.HTML)
	graphData := make(map[string]template.JS)

	probes := ps.probeNames
	if v := hw.r.URL.Query()["probe"]; v != nil {
		probes = v
	}
	maxDuration := time.Duration(ps.c.GetTimeseriesSize()) * ps.resolution
	graphOpts := graphOptsFromURL(hw.r.URL.Query(), maxDuration, ps.l)

	for _, probeName := range probes {
		statusTable[probeName] = template.HTML(ps.statusTable(probeName))
		debugData[probeName] = template.HTML(ps.debugLines(probeName))

		// Compute graph data and convert it into JSON for embedding.
		gd := computeGraphData(ps.metrics[probeName], graphOpts)
		graphData[probeName] = template.JS(gd.JSONBytes(ps.l))
	}

	var statusBuf bytes.Buffer

	err := statusTmpl.Execute(&statusBuf, struct {
		BaseURL     string
		Durations   []string
		ProbeNames  []string
		AllProbes   []string // Unfiltered probes
		StatusTable map[string]template.HTML
		GraphData   map[string]template.JS
		DebugData   map[string]template.HTML
		Header      template.HTML
		StartTime   fmt.Stringer
	}{
		BaseURL:     ps.c.GetUrl(),
		Durations:   ps.dashDurationsText,
		ProbeNames:  probes,
		AllProbes:   ps.probeNames,
		StatusTable: statusTable,
		GraphData:   graphData,
		DebugData:   debugData,
		Header:      resources.Header(),
		StartTime:   ps.startTime,
	})
	if err != nil {
		ps.l.Errorf("Error executing probe status template: %v", err)
		return
	}
	ps.pageCache.setContent(hw.r.URL.String(), statusBuf.Bytes())
	hw.w.Write(statusBuf.Bytes())
}
