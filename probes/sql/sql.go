// Copyright 2026 The Cloudprober Authors.
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

// Package sql implements a SQL database probe type. Each probe run
// establishes a new database connection and runs the configured query (or
// just pings the server if no query is configured), so probe latency covers
// the full path: connect + auth + query.
package sql

import (
	"bytes"
	"context"
	"crypto/tls"
	gosql "database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/file"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/sql/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	query         string
	tlsConfig     *tls.Config
	payloadParser *payload.Parser

	// openDB returns a database handle for the given target. It's a struct
	// member (set per flavor in Init) to allow overriding in tests.
	openDB func(target endpoint.Endpoint) (*gosql.DB, error)
}

type probeResult struct {
	total, success    int64
	latency           metrics.LatencyValue
	validationFailure *metrics.Map[int64]
	payloadMetrics    []*metrics.EventMetrics
}

func (p *Probe) newResult() sched.ProbeResult {
	result := &probeResult{}

	if p.opts.Validators != nil {
		result.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}

	if p.opts.LatencyDist != nil {
		result.latency = p.opts.LatencyDist.CloneDist()
	} else {
		result.latency = metrics.NewFloat(0)
	}

	return result
}

func (result *probeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(opts.LatencyMetricName, result.latency.Clone()).
		AddLabel("ptype", "sql")

	if result.validationFailure != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	ems := append([]*metrics.EventMetrics{em}, result.payloadMetrics...)
	result.payloadMetrics = nil

	return ems
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not sql probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	switch p.c.GetFlavor() {
	case configpb.ProbeConf_POSTGRES:
		p.openDB = p.pgOpenDB
	default:
		return fmt.Errorf("flavor is required; supported flavors: POSTGRES")
	}

	if p.c.GetQuery() != "" && p.c.GetQueryFile() != "" {
		return errors.New("only one of query and query_file can be set")
	}
	p.query = p.c.GetQuery()
	if f := p.c.GetQueryFile(); f != "" {
		b, err := file.ReadFile(context.Background(), f)
		if err != nil {
			return fmt.Errorf("reading query_file %q: %v", f, err)
		}
		p.query = string(b)
	}

	// A ping-only probe produces no response body, so validators would fail
	// every run against a healthy database.
	if p.opts.Validators != nil && p.query == "" {
		return errors.New("validators require a query; a ping-only probe has no response to validate")
	}

	if p.c.GetTlsConfig() != nil {
		p.tlsConfig = &tls.Config{}
		if err := tlsconfig.UpdateTLSConfig(p.tlsConfig, p.c.GetTlsConfig()); err != nil {
			return fmt.Errorf("tls_config error: %v", err)
		}
	}

	if p.c.GetResponseMetricsOptions() != nil {
		var err error
		p.payloadParser, err = payload.NewParser(p.c.GetResponseMetricsOptions(), p.l)
		if err != nil {
			return fmt.Errorf("error initializing response metrics parser: %v", err)
		}
	}

	return nil
}

// targetLabels returns the labels available for @label@ substitution in the
// connection string. Same set as the external probe's substitutions.
func (p *Probe) targetLabels(target endpoint.Endpoint) map[string]string {
	labels := map[string]string{
		"target":      target.Name,
		"target.name": target.Name,
		"target.port": strconv.Itoa(target.Port),
		"port":        strconv.Itoa(target.Port),
		"probe":       p.name,
	}
	if target.IP != nil {
		labels["target.ip"] = target.IP.String()
	}
	for lk, lv := range target.Labels {
		labels["target.label."+lk] = lv
	}
	return labels
}

// runQuery connects to the database and runs the configured query, returning
// the serialized query results. If no query is configured, it just pings the
// database server.
func (p *Probe) runQuery(ctx context.Context, target endpoint.Endpoint) ([]byte, error) {
	db, err := p.openDB(target)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if p.query == "" {
		return nil, db.PingContext(ctx)
	}

	rows, err := db.QueryContext(ctx, p.query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return serializeRows(rows)
}

// maxQueryResultSize caps how much serialized query output a single probe run
// can produce. Results are buffered in memory on every run, so a query
// returning a huge result set would otherwise create sustained memory
// pressure; exceeding the cap fails the probe with an error suggesting a
// LIMIT.
const maxQueryResultSize = 1 << 20 // 1MB

// serializeRows converts query results into text that validators and the
// payload metrics parser can consume: one line per row, with column values
// joined by a single space.
func serializeRows(rows *gosql.Rows) ([]byte, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	vals := make([]any, len(cols))
	for i := range vals {
		vals[i] = new(any)
	}

	var buf bytes.Buffer
	for rows.Next() {
		if err := rows.Scan(vals...); err != nil {
			return nil, err
		}
		for i, v := range vals {
			if i > 0 {
				buf.WriteByte(' ')
			}
			val := *(v.(*any))
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			fmt.Fprintf(&buf, "%v", val)
		}
		buf.WriteByte('\n')

		if buf.Len() > maxQueryResultSize {
			return nil, fmt.Errorf("query result exceeded %d bytes, consider adding a LIMIT to the query", maxQueryResultSize)
		}
	}
	return buf.Bytes(), rows.Err()
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}

	target, result := runReq.Target, runReq.Result.(*probeResult)
	l := p.l.WithAttributes(slog.String("target", target.Name))

	result.total++

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, "", target.Port)
	}

	start := time.Now()
	respBody, err := p.runQuery(ctx, target)
	latency := time.Since(start)

	if err != nil {
		l.Error(err.Error())
		runReq.LastRun.Set(false, 0, err)
		return
	}

	if p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: respBody}, result.validationFailure, l)

		// If any validation failed, return now, leaving the success and
		// latency counters unchanged.
		if len(failedValidations) > 0 {
			err := fmt.Errorf("failed validations: %s", strings.Join(failedValidations, ","))
			l.Error(err.Error())
			runReq.LastRun.Set(false, 0, err)
			return
		}
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())

	if p.payloadParser != nil {
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: respBody}, target.Dst()) {
			result.payloadMetrics = append(result.payloadMetrics, em.AddLabel("ptype", "sql"))
		}
	}

	runReq.LastRun.Set(true, latency, nil)
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running SQL probe once.")
	return sched.RunOnce(ctx, p.opts, p.runProbe)
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		RunProbeForTarget: p.runProbe,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
