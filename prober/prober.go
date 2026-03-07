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

/*
Package prober provides a prober for running a set of probes.

Prober takes in a config proto which dictates what probes should be created
with what configuration, and manages the asynchronous fan-in/fan-out of the
metrics data from these probes.
*/
package prober

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"slices"
	"sync"
	"time"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	rdsserver "github.com/cloudprober/cloudprober/internal/rds/server"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	spb "github.com/cloudprober/cloudprober/prober/proto"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/probes/options"
	probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/cloudprober/cloudprober/targets/lameduck"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	probesConfigSavePath = flag.String("probes_config_save_path", "", "Path to save the config to on API triggered config changes. If empty, config saving is disabled.")
)

// Prober represents a collection of probes where each probe implements the Probe interface.
type Prober struct {
	Probes    map[string]*probes.ProbeInfo
	Servers   []*servers.ServerInfo
	c         *configpb.ProberConfig
	l         *logger.Logger
	mu        sync.RWMutex
	ldLister  endpoint.Lister
	Surfacers []*surfacers.SurfacerInfo

	// We need this to start probes in response to API trigger. We still want
	// these probes to exit if prober's start context gets canceled.
	startCtx context.Context

	// Probe channel to handle starting of the new probes.
	grpcStartProbeCh chan string

	// Per-probe cancelFunc map.
	probeCancelFunc map[string]context.CancelFunc

	// dataChan for passing metrics between probes and main goroutine.
	dataChan chan *metrics.EventMetrics

	// Required for all gRPC server implementations.
	spb.UnimplementedCloudproberServer
}

func runOnThisHost(runOn string, hostname string) (bool, error) {
	if runOn == "" {
		return true, nil
	}
	r, err := regexp.Compile(runOn)
	if err != nil {
		return false, err
	}
	return r.MatchString(hostname), nil
}

func (pr *Prober) addProbe(p *probes_configpb.ProbeDef) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Check if this probe is supposed to run here.
	runHere, err := runOnThisHost(p.GetRunOn(), sysvars.GetVar("hostname"))
	if err != nil {
		return err
	}
	if !runHere {
		return nil
	}

	if pr.Probes[p.GetName()] != nil {
		return status.Errorf(codes.AlreadyExists, "probe %s is already defined", p.GetName())
	}

	opts, err := options.BuildProbeOptions(p, pr.ldLister, pr.c, pr.l)
	if err != nil {
		return status.Error(codes.Unknown, err.Error())
	}

	pr.l.Infof("Creating a %s probe: %s", p.GetType(), p.GetName())
	probeInfo, err := probes.CreateProbe(p, opts)
	if err != nil {
		return status.Error(codes.Unknown, err.Error())
	}
	pr.Probes[p.GetName()] = probeInfo

	return nil
}

// startProbe starts the probe with the given name.
// startProbe is protected and can be called concurrently. It's called
// from Start() at the very beginning, and then every time a new probe is
// added through a gRPC request.
func (pr *Prober) startProbe(name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	probeCtx, cancelFunc := context.WithCancel(pr.startCtx)
	pr.probeCancelFunc[name] = cancelFunc

	p := pr.Probes[name]
	go func() {
		if delay := p.ProbeDef.GetStartupDelayMsec(); delay > 0 {
			select {
			case <-probeCtx.Done():
				return
			case <-time.After(time.Duration(delay) * time.Millisecond):
			}
		}
		p.Start(probeCtx, pr.dataChan)
	}()
}

func randomDuration(duration time.Duration) time.Duration {
	if duration == 0 {
		return 0
	}
	if duration > time.Minute {
		duration = time.Minute
	}
	return time.Duration(rand.Int63n(duration.Milliseconds())) * time.Millisecond
}

// interProbeGap returns the wait time between probes that are in the same
// interval bucket.
//
// Overall goal is to start all probes by the end of 1st interval. But to avoid
// waiting for too long for large intervals, we cap the gap at 1 minute.
// Some examples:
//
//	30 probes in 30s interval, gap=1s, last probe starts at 29s
//	12 probes in 1m interval, gap=5s, last probe starts at 55s
//	12 probes in 5m interval, gap=25s, last probe starts at 4m35s
//	8 probes in 10m interval, gap=1m, last probe starts at 7m
//
// TODO(manugarg): Allow this to be overridden with config.
func interProbeGap(interval time.Duration, numProbes int) time.Duration {
	d := interval / time.Duration(numProbes)
	if d > 1*time.Minute {
		return 1 * time.Minute
	}
	return d
}

// startProbesWithJitter try to space out probes over time, as much as
// possible, without making it too complicated.
//
// We arrange probes into interval buckets - all probes with the same interval
// will be part of the same bucket. We spread out probes within an interval,
// and also the overall interval-buckets themselves.
//
//	[probe1 <gap> probe2 <gap> probe3 <gap> ...]    interval1 (30s)
//	<interval-bucket-gap> [probe4 <gap> probe5 ...] interval2 (10s)
//	<interval-bucket-gap> [probe6 <gap> probe7 ...] interval3 (1m)
//
// Note this function is not concurrency safe, which is fine, because it's only
// called from Start(), which is never called concurrently itself.
func (pr *Prober) startProbesWithJitter() {
	// Make interval -> [probe1, probe2, probe3..] map
	intervalBuckets := make(map[time.Duration][]*probes.ProbeInfo)

	for _, p := range pr.Probes {
		intervalBuckets[p.Options.Interval] = append(intervalBuckets[p.Options.Interval], p)
	}

	iter := 0

	for interval, probeInfos := range intervalBuckets {
		// Note that we introduce jitter within the goroutine instead of in the
		// loop here. This is to make sure this function returns quickly.
		go func(interval time.Duration, probeInfos []*probes.ProbeInfo, iter int) {
			if iter > 0 {
				// Except for the first interval bucket, we sleep before
				// starting probes in this bucket. This sleep is for a random
				// duration <= min(interval, 1min). Note that unlike inter-probe
				// gap this one doesn't add up.
				time.Sleep(randomDuration(interval))
			}

			for _, p := range probeInfos {
				pr.l.Info("Starting probe: ", p.Name)
				go pr.startProbe(p.Name)
				time.Sleep(interProbeGap(interval, len(probeInfos)))
			}
		}(interval, probeInfos, iter)
		iter++
	}
}

// Run runs requested 'probeNames' once.
func (pr *Prober) Run(ctx context.Context, probeNames []string) (map[string][]*singlerun.ProbeRunResult, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	for _, wanted := range probeNames {
		missing := make([]string, 0)
		if _, ok := pr.Probes[wanted]; !ok {
			missing = append(missing, wanted)
		}
		if len(missing) > 0 {
			return nil, status.Errorf(codes.NotFound, "some probes are not found: %v", missing)
		}
	}

	out := make(map[string][]*singlerun.ProbeRunResult)
	var outMu sync.Mutex

	var wg sync.WaitGroup
	for name := range pr.Probes {
		if len(probeNames) > 0 && !slices.Contains(probeNames, name) {
			continue
		}
		p, ok := pr.Probes[name].Probe.(probes.ProbeWithRunOnce)
		if !ok {
			pr.l.Warningf("probe %s doesn't support single run", name)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			prrs := p.RunOnce(ctx)

			outMu.Lock()
			defer outMu.Unlock()

			for _, prr := range prrs {
				if prr.Error != nil {
					pr.l.Errorf("probe (%s) failure for target %s: %v", name, prr.Target.Dst(), prr.Error)
				}
			}
			out[name] = append(out[name], prrs...)
		}()
	}
	wg.Wait()
	return out, nil
}

// Start starts a previously initialized Cloudprober.
// Start is unprotected and should never be called concurrently. It's called
// once from the main function, never in response to a gRPC request.
func (pr *Prober) Start(ctx context.Context) {
	pr.startCtx = ctx

	pr.dataChan = make(chan *metrics.EventMetrics, 100000)

	go func() {
		var em *metrics.EventMetrics
		for {
			em = <-pr.dataChan

			// Replicate the surfacer message to every surfacer we have
			// registered. Note that s.Write() is expected to be
			// non-blocking to avoid blocking of EventMetrics message
			// processing.
			for _, surfacer := range pr.Surfacers {
				surfacer.Write(pr.startCtx, em)
			}
		}
	}()

	// Start a goroutine to export system variables
	go sysvars.Start(ctx, pr.dataChan, time.Millisecond*time.Duration(pr.c.GetSysvarsIntervalMsec()), pr.c.GetSysvarsEnvVar())

	// Start servers, each in its own goroutine
	for _, s := range pr.Servers {
		go s.Start(ctx, pr.dataChan)
	}

	if pr.c.GetDisableJitter() {
		for name := range pr.Probes {
			go pr.startProbe(name)
		}
	} else {
		pr.startProbesWithJitter()
	}
}

// Init initialize prober with the given config file.
func Init(ctx context.Context, cfg *configpb.ProberConfig, l *logger.Logger) (*Prober, error) {
	pr := &Prober{
		c: cfg,
		l: l,
	}

	// Initialize cloudprober gRPC service if configured.
	srv := state.DefaultGRPCServer()
	if srv != nil {
		pr.grpcStartProbeCh = make(chan string, 10)
		spb.RegisterCloudproberServer(srv, pr)
	}

	// Initialize RDS server, if configured and attach to the default gRPC server.
	// Note that we can still attach services to the default gRPC server as it's
	// started later in Start().
	if c := pr.c.GetRdsServer(); c != nil {
		rdsServer, err := rdsserver.New(ctx, c, nil, logger.NewWithAttrs(slog.String("component", "rds-server")))
		if err != nil {
			return nil, err
		}

		state.SetLocalRDSServer(rdsServer)
		if srv != nil {
			rdsServer.RegisterWithGRPC(srv)
		}
	}

	// Initialize lameduck lister
	globalTargetsOpts := pr.c.GetGlobalTargetsOptions()

	if globalTargetsOpts.GetLameDuckOptions() != nil {
		ldLogger := logger.NewWithAttrs(slog.String("component", "lame-duck"))

		if err := lameduck.InitDefaultLister(globalTargetsOpts, nil, ldLogger); err != nil {
			return nil, err
		}

		var err error
		pr.ldLister, err = lameduck.GetDefaultLister()
		if err != nil {
			pr.l.Warningf("Error while getting default lameduck lister, lameduck behavior will be disabled. Err: %v", err)
		}
	}

	var err error

	// Initialize shared targets
	for _, st := range pr.c.GetSharedTargets() {
		tgts, err := targets.New(st.GetTargets(), pr.ldLister, globalTargetsOpts, pr.l, pr.l)
		if err != nil {
			return nil, err
		}
		targets.SetSharedTargets(st.GetName(), tgts)
	}

	// Initiliaze probes
	pr.Probes = make(map[string]*probes.ProbeInfo)
	pr.probeCancelFunc = make(map[string]context.CancelFunc)
	for _, p := range pr.c.GetProbe() {
		if err := pr.addProbe(p); err != nil {
			return nil, fmt.Errorf("error while adding probe '%s': %v", p.GetName(), err)
		}
	}

	// Initialize servers
	pr.Servers, err = servers.Init(ctx, pr.c.GetServer())
	if err != nil {
		return nil, fmt.Errorf("error while initializing servers: %v", err)
	}

	pr.Surfacers, err = surfacers.Init(ctx, pr.c.GetSurfacer())
	if err != nil {
		return nil, fmt.Errorf("error while initializing surfacers: %v", err)
	}

	return pr, nil
}
