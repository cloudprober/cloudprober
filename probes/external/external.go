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

/*
Package external implements an external probe type for cloudprober.

External probe type executes an external process for actual probing. These probes
can have two modes: "once" and "server". In "once" mode, the external process is
started for each probe run cycle, while in "server" mode, external process is
started only if it's not running already and Cloudprober communicates with it
over stdin/stdout for each probe cycle.
*/
package external

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	payloadpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/cloudprober/cloudprober/probes/common/command"
	configpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/google/shlex"
)

var (
	validLabelRe = regexp.MustCompile(`@(target|address|port|probe|target.(name|ip|port)|target\.label\.[^@]+)@`)
)

type result struct {
	total, success    int64
	latency           metrics.LatencyValue
	validationFailure *metrics.Map[int64]
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	cmdName string
	cmdArgs []string
	envVars []string
	opts    *options.Options
	c       *configpb.ProbeConf
	l       *logger.Logger

	// book-keeping params
	labelKeys    map[string]bool // Labels for substitution
	requestID    int32
	cmdRunning   bool
	cmdRunningMu sync.Mutex // synchronizes cmdRunning
	cmdStdin     io.Writer
	cmdStdout    io.ReadCloser
	cmdStderr    io.ReadCloser
	replyChan    chan *configpb.ProbeReply
	targets      []endpoint.Endpoint
	results      map[string]*result // probe results keyed by targets
	dataChan     chan *metrics.EventMetrics

	// default payload metrics that we clone from to build per-target payload
	// metrics.
	payloadParser *payload.Parser
}

func (p *Probe) updateLabelKeys() {
	p.labelKeys = make(map[string]bool)

	updateLabelKeysFn := func(s string) {
		matches := validLabelRe.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				// Pick the match within outer parentheses.
				p.labelKeys[m[1]] = true
			}
		}
	}

	for _, opt := range p.c.GetOptions() {
		updateLabelKeysFn(opt.GetValue())
	}
	for _, arg := range p.cmdArgs {
		updateLabelKeysFn(arg)
	}
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not external probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c
	p.replyChan = make(chan *configpb.ProbeReply)

	cmdParts, err := shlex.Split(p.c.GetCommand())
	if err != nil {
		return fmt.Errorf("error parsing command line (%s): %v", p.c.GetCommand(), err)
	}

	if len(cmdParts) == 0 {
		return errors.New("command not specified")
	}

	p.cmdName = cmdParts[0]
	p.cmdArgs = cmdParts[1:]

	for k, v := range p.c.GetEnvVar() {
		if v == "" {
			v = "1" // default to a truthy value
		}
		p.envVars = append(p.envVars, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(p.envVars)

	// Figure out labels we are interested in
	p.updateLabelKeys()

	p.results = make(map[string]*result)

	if !p.c.GetOutputAsMetrics() {
		return nil
	}

	omo := p.c.GetOutputMetricsOptions()
	if omo.GetMetricsKind() == payloadpb.OutputMetricsOptions_UNDEFINED && p.c.GetMode() == configpb.ProbeConf_SERVER {
		if omo == nil {
			omo = &payloadpb.OutputMetricsOptions{}
		}
		omo.MetricsKind = payloadpb.OutputMetricsOptions_CUMULATIVE.Enum()
	}
	p.payloadParser, err = payload.NewParser(omo, p.l)
	if err != nil {
		return fmt.Errorf("error initializing payload metrics: %v", err)
	}

	return nil
}

type commandIntf interface {
	Wait() error
}

func (p *Probe) labels(ep endpoint.Endpoint) map[string]string {
	labels := make(map[string]string)

	for k := range p.labelKeys {
		if v, ok := map[string]string{
			"target":      ep.Name,
			"target.name": ep.Name,
			"port":        strconv.Itoa(ep.Port),
			"target.port": strconv.Itoa(ep.Port),
			"target.ip":   ep.IP.String(),
			"probe":       p.name,
		}[k]; ok {
			labels[k] = v
		}
	}

	if p.labelKeys["address"] {
		addr, err := p.opts.Targets.Resolve(ep.Name, p.opts.IPVersion)
		if err != nil {
			p.l.Warningf("Targets.Resolve(%v, %v) failed: %v ", ep.Name, p.opts.IPVersion, err)
		} else if !addr.IsUnspecified() {
			labels["address"] = addr.String()
		}
	}
	for lk, lv := range ep.Labels {
		k := "target.label." + lk
		if p.labelKeys[k] {
			labels[k] = lv
		}
	}
	return labels
}

func (p *Probe) withStdLabels(em *metrics.EventMetrics, target endpoint.Endpoint) *metrics.EventMetrics {
	return em.AddLabel("ptype", "external").AddLabel("probe", p.name).AddLabel("dst", target.Dst())
}

type probeStatus struct {
	success bool
	latency time.Duration
	payload string
}

func (p *Probe) processProbeResult(ps *probeStatus, target endpoint.Endpoint, result *result) {
	if ps.success && p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: []byte(ps.payload)}, result.validationFailure, p.l)

		// If any validation failed, log and set success to false.
		if len(failedValidations) > 0 {
			p.l.Debug("Target:", target.Name, " failed validations: ", strings.Join(failedValidations, ","), ".")
			ps.success = false
		}
	}

	if ps.success {
		result.success++
		result.latency.AddFloat64(ps.latency.Seconds() / p.opts.LatencyUnit.Seconds())
	}

	defaultEM := metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric(p.opts.LatencyMetricName, result.latency.Clone())

	if p.opts.Validators != nil {
		defaultEM.AddMetric("validation_failure", result.validationFailure)
	}

	p.opts.RecordMetrics(target, p.withStdLabels(defaultEM, target), p.dataChan)

	// If probe is configured to use the external process output (or reply payload
	// in case of server probe) as metrics.
	if p.c.GetOutputAsMetrics() {
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: []byte(ps.payload)}, target.Dst()) {
			p.opts.RecordMetrics(target, p.withStdLabels(em, target), p.dataChan, options.WithNoAlert())
		}
	}
}

func (p *Probe) runOnceProbe(ctx context.Context) {
	var wg sync.WaitGroup

	for _, target := range p.targets {
		wg.Add(1)
		go func(target endpoint.Endpoint, result *result) {
			defer wg.Done()

			args := append([]string{}, p.cmdArgs...)
			if len(p.labelKeys) != 0 {
				for i, arg := range p.cmdArgs {
					res, found := strtemplate.SubstituteLabels(arg, p.labels(target))
					if !found {
						p.l.Warningf("Substitution not found in %q", arg)
					}
					args[i] = res
				}
			}

			p.l.Infof("Running external command: %s %s", p.cmdName, strings.Join(args, " "))
			result.total++

			cmd := &command.Command{
				CmdLine: append([]string{p.cmdName}, args...),
				EnvVars: p.envVars,
			}
			if p.c.GetOutputAsMetrics() && !p.c.GetDisableStreamingOutputMetrics() {
				cmd.ProcessStreamingOutput = func(line []byte) {
					for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: line}, target.Dst()) {
						p.opts.RecordMetrics(target, p.withStdLabels(em, target), p.dataChan, options.WithNoAlert())
					}
				}
			}

			startTime := time.Now()
			stdout, err := cmd.Execute(ctx, p.l)
			latency := time.Since(startTime)
			if err != nil {
				p.l.Errorf("Error running external probe: %v", err)
			}
			p.processProbeResult(&probeStatus{success: err == nil, latency: latency, payload: stdout}, target, result)
		}(target, p.results[target.Key()])
	}
	wg.Wait()
}

func (p *Probe) updateTargets() {
	p.targets = p.opts.Targets.ListEndpoints()

	for _, target := range p.targets {
		if _, ok := p.results[target.Key()]; ok {
			continue
		}

		var latencyValue metrics.LatencyValue
		if p.opts.LatencyDist != nil {
			latencyValue = p.opts.LatencyDist.CloneDist()
		} else {
			latencyValue = metrics.NewFloat(0)
		}

		p.results[target.Key()] = &result{
			latency:           latencyValue,
			validationFailure: validators.ValidationFailureMap(p.opts.Validators),
		}

		for _, al := range p.opts.AdditionalLabels {
			al.UpdateForTarget(target, "", 0)
		}
	}
}

func (p *Probe) runProbe(startCtx context.Context) {
	probeCtx, cancelFunc := context.WithTimeout(startCtx, p.opts.Timeout)
	defer cancelFunc()

	p.updateTargets()

	if p.c.GetMode() == configpb.ProbeConf_SERVER {
		p.runServerProbe(probeCtx, startCtx)
	} else {
		p.runOnceProbe(probeCtx)
	}
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(startCtx context.Context, dataChan chan *metrics.EventMetrics) {
	p.dataChan = dataChan

	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		select {
		case <-startCtx.Done():
			return
		default:
		}

		if !p.opts.IsScheduled() {
			continue
		}

		p.runProbe(startCtx)
	}
}
