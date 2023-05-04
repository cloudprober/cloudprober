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
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"flag"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/common/file"
	"github.com/cloudprober/cloudprober/config"
	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/sysvars"
	"github.com/cloudprober/cloudprober/web"
	"github.com/golang/glog"
)

var (
	configFile       = flag.String("config_file", "", "Config file")
	versionFlag      = flag.Bool("version", false, "Print version and exit")
	buildInfoFlag    = flag.Bool("buildinfo", false, "Print build info and exit")
	stopTime         = flag.Duration("stop_time", 0, "How long to wait for cleanup before process exits on SIGINT and SIGTERM")
	cpuprofile       = flag.String("cpuprof", "", "Write cpu profile to file")
	memprofile       = flag.String("memprof", "", "Write heap profile to file")
	configTest       = flag.Bool("configtest", false, "Dry run to test config file")
	dumpConfig       = flag.Bool("dumpconfig", false, "Dump processed config to stdout")
	testInstanceName = flag.String("test_instance_name", "ig-us-central1-a-01-0000", "Instance name example to be used in tests")

	// configTestVars provides a sane set of sysvars for config testing.
	configTestVars = map[string]string(nil)
)

// These variables get overwritten by using -ldflags="-X main.<var>=<value?" at
// the build time.
var version string
var buildTimestamp string
var dirty string

func setupConfigTestVars() {
	configTestVars = map[string]string{
		"zone":              "us-central1-a",
		"project":           "fake-domain.com:fake-project",
		"project_id":        "12345678",
		"instance":          *testInstanceName,
		"internal_ip":       "192.168.0.10",
		"external_ip":       "10.10.10.10",
		"instance_template": "ig-us-central1-a-01",
		"machine_type":      "e2-small",
	}
}

const (
	configMetadataKeyName = "cloudprober_config"
	defaultConfigFile     = "/etc/cloudprober.cfg"
)

func setupProfiling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	var f *os.File
	if *cpuprofile != "" {
		var err error
		f, err = os.Create(*cpuprofile)
		if err != nil {
			glog.Exit(err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			glog.Errorf("Could not start CPU profiling: %v", err)
		}
	}
	go func(file *os.File) {
		<-sigChan
		pprof.StopCPUProfile()
		if *cpuprofile != "" {
			if err := file.Close(); err != nil {
				glog.Exit(err)
			}
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				glog.Exit(err)
			}
			if err = pprof.WriteHeapProfile(f); err != nil {
				glog.Exit(err)
			}
			if err := f.Close(); err != nil {
				glog.Exit(err)
			}
		}
		os.Exit(1)
	}(f)
}

func configFileToString(fileName string) string {
	b, err := file.ReadFile(fileName)
	if err != nil {
		glog.Exitf("Failed to read the config file: %v", err)
	}
	return string(b)
}

func getConfig() string {
	if *configFile != "" {
		return configFileToString(*configFile)
	}
	// On GCE first check if there is a config in custom metadata
	// attributes.
	if metadata.OnGCE() {
		if config, err := config.ReadFromGCEMetadata(configMetadataKeyName); err != nil {
			glog.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config
		}
	}
	// If config not found in metadata, check default config on disk
	if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
		return configFileToString(defaultConfigFile)
	}
	glog.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return config.DefaultConfig()
}

func main() {
	flag.Parse()

	if len(flag.Args()) > 0 {
		glog.Exitf("Unexpected non-flag arguments: %v", flag.Args())
	}

	if dirty == "1" {
		version = version + " (dirty)"
	}

	runconfig.SetVersion(version)
	if buildTimestamp != "" {
		ts, err := strconv.ParseInt(buildTimestamp, 10, 64)
		if err != nil {
			glog.Exitf("Error parsing build timestamp (%s). Err: %v", buildTimestamp, err)
		}
		runconfig.SetBuildTimestamp(time.Unix(ts, 0))
	}

	if *versionFlag {
		fmt.Println(runconfig.Version())
		return
	}

	if *buildInfoFlag {
		fmt.Println(runconfig.Version())
		fmt.Println("Built at: ", runconfig.BuildTimestamp())
		return
	}

	setupConfigTestVars()

	if *dumpConfig {
		sysvars.Init(nil, configTestVars)
		text, err := config.ParseTemplate(getConfig(), sysvars.Vars())
		if err != nil {
			glog.Exitf("Error parsing config file. Err: %v", err)
		}
		fmt.Println(text)
		return
	}

	if *configTest {
		sysvars.Init(nil, configTestVars)
		_, err := config.ParseForTest(configFileToString(*configFile), sysvars.Vars())
		if err != nil {
			glog.Exitf("Error parsing config file. Err: %v", err)
		}
		return
	}

	setupProfiling()

	if err := cloudprober.InitFromConfig(getConfig()); err != nil {
		glog.Exitf("Error initializing cloudprober. Err: %v", err)
	}

	// web.Init sets up web UI for cloudprober.
	if err := web.Init(); err != nil {
		glog.Exitf("Error initializing web interface. Err: %v", err)
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
			glog.Warningf("Received signal \"%v\", canceling the start context and waiting for %v before closing", sig, *stopTime)
			cancelF()
			time.Sleep(*stopTime)
			os.Exit(0)
		}()
	}
	cloudprober.Start(startCtx)

	// Wait forever
	select {}
}
