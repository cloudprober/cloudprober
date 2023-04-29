// Copyright 2017-2020 The Cloudprober Authors.
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
Package options provides a shared interface to common probe options.
*/
package options

import (
	"fmt"
	"net"
	"time"

	"github.com/cloudprober/cloudprober/common/iputils"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/alerting"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/cloudprober/cloudprober/validators"
)

// Options encapsulates common probe options.
type Options struct {
	Targets             targets.Targets
	Interval, Timeout   time.Duration
	Logger              *logger.Logger
	ProbeConf           interface{} // Probe-type specific config
	LatencyDist         *metrics.Distribution
	LatencyUnit         time.Duration
	LatencyMetricName   string
	Validators          []*validators.Validator
	SourceIP            net.IP
	IPVersion           int
	StatsExportInterval time.Duration
	LogMetrics          func(*metrics.EventMetrics)
	AdditionalLabels    []*AdditionalLabel
	NegativeTest        bool
	AlertHandlers       []*alerting.AlertHandler
}

const defaultStatsExtportIntv = 10 * time.Second
const defaultIntervalPeriod = 2 * time.Second
const defaultTimeoutPeriod = 1 * time.Second

var negativeTestSupported = map[configpb.ProbeDef_Type]bool{
	configpb.ProbeDef_TCP:  true,
	configpb.ProbeDef_PING: true,
}

func defaultStatsExportInterval(p *configpb.ProbeDef, opts *Options) time.Duration {
	minIntv := opts.Interval
	if opts.Timeout > opts.Interval {
		minIntv = opts.Timeout
	}

	// UDP probe type requires stats export interval to be at least twice of the
	// max(interval, timeout).
	if p.GetType() == configpb.ProbeDef_UDP {
		minIntv = 2 * minIntv
	}

	if minIntv < defaultStatsExtportIntv {
		return defaultStatsExtportIntv
	}
	return minIntv
}

func ipv(v *configpb.ProbeDef_IPVersion) int {
	if v == nil {
		return 0
	}

	switch *v {
	case configpb.ProbeDef_IPV4:
		return 4
	case configpb.ProbeDef_IPV6:
		return 6
	default:
		return 0
	}
}

// getSourceFromConfig returns the source IP from the config either directly
// or by resolving the network interface to an IP, depending on which is provided.
func getSourceIPFromConfig(p *configpb.ProbeDef, l *logger.Logger) (net.IP, error) {
	switch p.SourceIpConfig.(type) {

	case *configpb.ProbeDef_SourceIp:
		sourceIP := net.ParseIP(p.GetSourceIp())
		if sourceIP == nil {
			return nil, fmt.Errorf("invalid source IP: %s", p.GetSourceIp())
		}

		// If ip_version is configured, make sure source_ip matches it.
		if ipv(p.IpVersion) != 0 && iputils.IPVersion(sourceIP) != ipv(p.IpVersion) {
			return nil, fmt.Errorf("configured source_ip (%s) doesn't match the ip_version (%d)", p.GetSourceIp(), ipv(p.IpVersion))
		}

		return sourceIP, nil

	case *configpb.ProbeDef_SourceInterface:
		return iputils.ResolveIntfAddr(p.GetSourceInterface(), ipv(p.IpVersion))

	default:
		return nil, fmt.Errorf("unknown source type: %v", p.GetSourceIpConfig())
	}
}

// BuildProbeOptions builds probe's options using the provided config and some
// global params.
func BuildProbeOptions(p *configpb.ProbeDef, ldLister endpoint.Lister, globalTargetsOpts *targetspb.GlobalTargetsOptions, l *logger.Logger) (*Options, error) {
	intervalDuration := defaultIntervalPeriod
	timeoutDuration := defaultTimeoutPeriod
	var err error

	if p.GetIntervalMsec() != 0 && p.GetInterval() != "" {
		return nil, fmt.Errorf("both interval (%s) and interval_msec (%d) are specified", p.GetInterval(), p.GetIntervalMsec())
	} else if p.GetIntervalMsec() != 0 {
		intervalDuration = time.Duration(p.GetIntervalMsec()) * time.Millisecond
	} else if p.GetInterval() != "" {
		intervalDuration, err = time.ParseDuration(p.GetInterval())
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval (%s): %v", p.GetInterval(), err)
		}
	}

	if p.GetTimeoutMsec() != 0 && p.GetTimeout() != "" {
		return nil, fmt.Errorf("both timeout (%s) and timeout_msec (%d) are specified", p.GetTimeout(), p.GetTimeoutMsec())
	} else if p.GetTimeoutMsec() != 0 {
		timeoutDuration = time.Duration(p.GetTimeoutMsec()) * time.Millisecond
	} else if p.GetTimeout() != "" {
		timeoutDuration, err = time.ParseDuration(p.GetTimeout())
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout (%s): %v", p.GetTimeout(), err)
		}
	}

	if p.GetNegativeTest() && !negativeTestSupported[p.GetType()] {
		return nil, fmt.Errorf("negative_test is not supported by %s probes", p.GetType().String())
	}

	opts := &Options{
		Interval:          intervalDuration,
		Timeout:           timeoutDuration,
		IPVersion:         ipv(p.IpVersion),
		LatencyMetricName: p.GetLatencyMetricName(),
		NegativeTest:      p.GetNegativeTest(),
	}

	if opts.Logger, err = logger.NewCloudproberLog(p.GetName()); err != nil {
		return nil, fmt.Errorf("error in initializing logger for the probe (%s): %v", p.GetName(), err)
	}

	if opts.Targets, err = targets.New(p.GetTargets(), ldLister, globalTargetsOpts, l, opts.Logger); err != nil {
		return nil, err
	}

	if latencyDist := p.GetLatencyDistribution(); latencyDist != nil {
		var d *metrics.Distribution
		if d, err = metrics.NewDistributionFromProto(latencyDist); err != nil {
			return nil, fmt.Errorf("error creating distribution from the specification (%v): %v", latencyDist, err)
		}
		opts.LatencyDist = d
	}

	// latency_unit is specified as a human-readable string, e.g. ns, ms, us etc.
	if opts.LatencyUnit, err = time.ParseDuration("1" + p.GetLatencyUnit()); err != nil {
		return nil, fmt.Errorf("failed to parse the latency unit (%s): %v", p.GetLatencyUnit(), err)
	}

	if len(p.GetValidator()) > 0 {
		opts.Validators, err = validators.Init(p.GetValidator(), opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize validators: %v", err)
		}
	}

	if p.GetSourceIpConfig() != nil {
		opts.SourceIP, err = getSourceIPFromConfig(p, l)
		if err != nil {
			return nil, fmt.Errorf("failed to get source address for the probe: %v", err)
		}
		// Set IPVersion from SourceIP if not already set.
		if opts.IPVersion == 0 {
			opts.IPVersion = iputils.IPVersion(opts.SourceIP)
		}
	}

	if p.StatsExportIntervalMsec == nil {
		opts.StatsExportInterval = defaultStatsExportInterval(p, opts)
	} else {
		opts.StatsExportInterval = time.Duration(p.GetStatsExportIntervalMsec()) * time.Millisecond
		if opts.StatsExportInterval < opts.Interval {
			return nil, fmt.Errorf("stats_export_interval (%d ms) smaller than probe interval %v", p.GetStatsExportIntervalMsec(), opts.Interval)
		}
	}

	opts.AdditionalLabels = parseAdditionalLabels(p)

	for _, alertConf := range p.GetAlert() {
		ah, err := alerting.NewAlertHandler(alertConf, p.GetName(), opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("error in initializing alert handler: %v", err)
		}
		opts.AlertHandlers = append(opts.AlertHandlers, ah)
	}

	if !p.GetDebugOptions().GetLogMetrics() {
		opts.LogMetrics = func(em *metrics.EventMetrics) {}
	} else {
		opts.LogMetrics = func(em *metrics.EventMetrics) {
			if opts.Logger != nil {
				opts.Logger.Info(em.String())
			}
		}
	}

	return opts, nil
}

// DefaultOptions returns default options, capturing default values for the
// various fields.
func DefaultOptions() *Options {
	p := &configpb.ProbeDef{
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_DummyTargets{},
		},
	}

	opts, err := BuildProbeOptions(p, nil, nil, nil)
	// Without no user input, there should be no errors. We execute this as part
	// of the tests.
	if err != nil {
		panic(err)
	}

	return opts
}
