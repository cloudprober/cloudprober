// Copyright 2017-2023 The Cloudprober Authors.
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
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	rdsserver "github.com/cloudprober/cloudprober/internal/rds/server"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
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
	"google.golang.org/protobuf/encoding/prototext"
)

var randGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

var (
	probesConfigSavePath = flag.String("probes_config_save_path", "", "Path to save the config to on API triggered config changes. If empty, config saving is disabled.")
)

// Prober represents a collection of probes where each probe implements the Probe interface.
type Prober struct {
	Probes    map[string]*probes.ProbeInfo
	Servers   []*servers.ServerInfo
	c         *configpb.ProberConfig
	l         *logger.Logger
	mu        sync.Mutex
	ldLister  endpoint.Lister
	Surfacers []*surfacers.SurfacerInfo

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

	opts, err := options.BuildProbeOptions(p, pr.ldLister, pr.c.GetGlobalTargetsOptions(), pr.l)
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}

	pr.l.Infof("Creating a %s probe: %s", p.GetType(), p.GetName())
	probeInfo, err := probes.CreateProbe(p, opts)
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	pr.Probes[p.GetName()] = probeInfo

	return nil
}

// Init initialize prober with the given config file.
func (pr *Prober) Init(ctx context.Context, cfg *configpb.ProberConfig, l *logger.Logger) error {
	pr.c = cfg
	pr.l = l

	// Initialize cloudprober gRPC service if configured.
	srv := state.DefaultGRPCServer()
	if srv != nil {
		pr.grpcStartProbeCh = make(chan string)
		spb.RegisterCloudproberServer(srv, pr)
	}

	// Initialize RDS server, if configured and attach to the default gRPC server.
	// Note that we can still attach services to the default gRPC server as it's
	// started later in Start().
	if c := pr.c.GetRdsServer(); c != nil {
		rdsServer, err := rdsserver.New(ctx, c, nil, logger.NewWithAttrs(slog.String("component", "rds-server")))
		if err != nil {
			return err
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
			return err
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
			return err
		}
		targets.SetSharedTargets(st.GetName(), tgts)
	}

	// Initiliaze probes
	pr.Probes = make(map[string]*probes.ProbeInfo)
	pr.probeCancelFunc = make(map[string]context.CancelFunc)
	for _, p := range pr.c.GetProbe() {
		if err := pr.addProbe(p); err != nil {
			return err
		}
	}

	// Initialize servers
	pr.Servers, err = servers.Init(ctx, pr.c.GetServer())
	if err != nil {
		return err
	}

	pr.Surfacers, err = surfacers.Init(ctx, pr.c.GetSurfacer())
	if err != nil {
		return err
	}

	return nil
}

// Start starts a previously initialized Cloudprober.
func (pr *Prober) Start(ctx context.Context) {
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
				surfacer.Write(context.Background(), em)
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
			go pr.startProbe(ctx, name)
		}
	} else {
		pr.startProbesWithJitter(ctx)
	}
	if state.DefaultGRPCServer() != nil {
		// Start a goroutine to handle starting of the probes added through gRPC.
		// AddProbe adds new probes to the pr.grpcStartProbeCh channel and this
		// goroutine reads from that channel and starts the probe using the overall
		// Start context.
		go func() {
			for {
				select {
				case name := <-pr.grpcStartProbeCh:
					pr.startProbe(ctx, name)
				}
			}
		}()
	}
}

func (pr *Prober) startProbe(ctx context.Context, name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	probeCtx, cancelFunc := context.WithCancel(ctx)
	pr.probeCancelFunc[name] = cancelFunc
	go pr.Probes[name].Start(probeCtx, pr.dataChan)
}

func randomDuration(duration, ceiling time.Duration) time.Duration {
	if duration == 0 {
		return 0
	}
	if duration > ceiling {
		duration = ceiling
	}
	return time.Duration(randGenerator.Int63n(duration.Milliseconds())) * time.Millisecond
}

// interProbeWait returns the wait time between probes. It's not beneficial for
// this interval to be too large, so we cap it at 2 seconds.
func interProbeWait(interval time.Duration, numProbes int) time.Duration {
	d := interval / time.Duration(numProbes)
	if d > 2*time.Second {
		return 2 * time.Second
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
func (pr *Prober) startProbesWithJitter(ctx context.Context) {
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
				time.Sleep(randomDuration(interval, 10*time.Second))
			}

			for _, p := range probeInfos {
				pr.l.Info("Starting probe: ", p.Name)
				go pr.startProbe(ctx, p.Name)
				time.Sleep(interProbeWait(interval, len(probeInfos)))
			}
		}(interval, probeInfos, iter)
		iter++
	}
}

func (pr *Prober) saveProbesConfigUnprotected(filePath string) error {
	var keys []string
	for k := range pr.Probes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cfg := &configpb.ProberConfig{}
	for _, k := range keys {
		cfg.Probe = append(cfg.Probe, pr.Probes[k].ProbeDef)
	}

	textCfg := prototext.MarshalOptions{
		Indent: "  ",
	}.Format(cfg)

	if textCfg == "" && len(pr.Probes) != 0 {
		err := fmt.Errorf("text marshaling of probes config returned an empty string. Config: %v", cfg)
		pr.l.Warning(err.Error())
		return err
	}

	if err := os.WriteFile(filePath, []byte(textCfg), 0644); err != nil {
		pr.l.Errorf("Error saving config to disk: %v", err)
		return err
	}

	return nil
}
