// Copyright 2017 The Cloudprober Authors.
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
Binary cloudprober is a tool for running a set of probes and metric surfacers
on a GCE VM. Cloudprober takes in a config proto which dictates what probes
and surfacers should be created with what configuration, and then manages the
asynchronous fan-in/fan-out of the data between the probes and the surfacers.
*/
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"flag"

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/config"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/state"
)

var (
	versionFlag       = flag.Bool("version", false, "Print version and exit")
	buildInfoFlag     = flag.Bool("buildinfo", false, "Print build info and exit")
	stopTime          = flag.Duration("stop_time", 0, "How long to wait for cleanup before process exits on SIGINT and SIGTERM")
	cpuprofile        = flag.String("cpuprof", "", "Write cpu profile to file")
	memprofile        = flag.String("memprof", "", "Write heap profile to file")
	configTest        = flag.Bool("configtest", false, "Dry run to test config file")
	dumpConfig        = flag.Bool("dumpconfig", false, "Dump processed config to stdout")
	dumpConfigFormat  = flag.String("dumpconfig_fmt", "textpb", "Dump config format (textpb, json, yaml)")
	runOnce           = flag.Bool("run_once", false, "Run a single probe and exit")
	runOnceProbeNames = flag.String("run_once_probe_names", "", "Comma-separated list of probe names to run")
	runOnceOutFormat  = flag.String("run_once_output_format", "text", "Run once output format (text, json)")
	runOnceOutIndent  = flag.String("run_once_output_indent", "  ", "Run once output indent")
)

// These variables get overwritten by using -ldflags="-X main.<var>=<value?" at
// the build time.
var version string
var buildTimestamp string
var dirty string
var l *logger.Logger

func setupProfiling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	var f *os.File
	if *cpuprofile != "" {
		var err error
		f, err = os.Create(*cpuprofile)
		if err != nil {
			l.Critical(err.Error())
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			l.Criticalf("Could not start CPU profiling: %v", err)
		}
	}
	go func(file *os.File) {
		<-sigChan
		pprof.StopCPUProfile()
		if *cpuprofile != "" {
			if err := file.Close(); err != nil {
				l.Critical(err.Error())
			}
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				l.Critical(err.Error())
			}
			if err = pprof.WriteHeapProfile(f); err != nil {
				l.Critical(err.Error())
			}
			if err := f.Close(); err != nil {
				l.Critical(err.Error())
			}
		}
		os.Exit(1)
	}(f)
}

func main() {
	flag.Parse()

	// Initialize logger after parsing flags.
	l = logger.NewWithAttrs(slog.String("component", "global"))

	if len(flag.Args()) > 0 {
		l.Criticalf("Unexpected non-flag arguments: %v", flag.Args())
	}

	if dirty == "1" {
		version = version + " (dirty)"
	}

	state.SetVersion(version)
	if buildTimestamp != "" {
		ts, err := strconv.ParseInt(buildTimestamp, 10, 64)
		if err != nil {
			l.Criticalf("Error parsing build timestamp (%s). Err: %v", buildTimestamp, err)
		}
		state.SetBuildTimestamp(time.Unix(ts, 0))
	}

	if *versionFlag {
		fmt.Println(state.Version())
		return
	}

	if *buildInfoFlag {
		fmt.Println(state.Version())
		fmt.Println("Built at: ", state.BuildTimestamp())
		return
	}

	if *dumpConfig {
		out, err := config.DumpConfig(*dumpConfigFormat, nil)
		if err != nil {
			l.Criticalf("Error dumping config. Err: %v", err)
		}
		fmt.Println(string(out))
		return
	}

	if *configTest {
		if err := config.ConfigTest(nil); err != nil {
			l.Criticalf("Config test failed. Err: %v", err)
		}
		return
	}

	setupProfiling()

	if err := cloudprober.Init(); err != nil {
		l.Criticalf("Error initializing cloudprober. Err: %v", err)
	}

	startCtx := context.Background()

	if *stopTime == 0 {
		*stopTime = time.Duration(cloudprober.GetConfig().GetStopTimeSec()) * time.Second
	}

	if *stopTime != 0 {
		// Set up signal handling for the cancelation of the start context.
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		ctx, cancelF := context.WithCancel(startCtx)
		startCtx = ctx

		go func() {
			sig := <-sigs
			l.Warningf("Received signal \"%v\", canceling the start context and waiting for %v before closing", sig, *stopTime)
			cancelF()
			time.Sleep(*stopTime)
			os.Exit(0)
		}()
	}

	if *runOnce {
		err := cloudprober.RunOnce(startCtx, *runOnceProbeNames, *runOnceOutFormat, *runOnceOutIndent)
		if err != nil {
			l.Criticalf("Error running run-once probe. Err: %v", err)
		}
		return
	}
	cloudprober.Start(startCtx)

	// Wait forever
	select {}
}
