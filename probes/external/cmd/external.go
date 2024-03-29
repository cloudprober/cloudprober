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

// This program implements a stand-alone external prober binary using the
// cloudprober/probes/external package. It's intended to help prototype the external package
// quickly and doesn't provide the facilities that cloudprober provides.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"flag"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/external"
	configpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	config    = flag.String("config_file", "", "Config file (ProbeConf)")
	command   = flag.String("command", "", "External probe command")
	intervalF = flag.Duration("interval", 60*time.Second, "Interval between probes")
	timeoutF  = flag.Duration("timeout", 59*time.Second, "Per-probe timeout")
	targetsF  = flag.String("targets", "", "Comma separated list of targets (optional). Useful if your probe config makes use of the @target@ or @address@ labels. See config protobuf for more on these labels.")
)

func main() {
	flag.Parse()

	c := &configpb.ProbeConf{
		Command: command,
	}

	// If we are given a config file, read it. If not, use defaults.
	if *config != "" {
		b, err := os.ReadFile(*config)
		if err != nil {
			log.Fatal(err)
		}
		if err = prototext.Unmarshal(b, c); err != nil {
			log.Fatalf("Error while parsing config protobuf %s: Err: %v", string(b), err)
		}
	}

	opts := &options.Options{
		Targets:   targets.StaticTargets(*targetsF),
		Interval:  *intervalF,
		Timeout:   *timeoutF,
		ProbeConf: c,
	}

	ep := &external.Probe{}
	if err := ep.Init("external_test", opts); err != nil {
		log.Fatalf("Error in initializing probe %s from the config. Err: %v", "external_test", err)
	}
	dataChan := make(chan *metrics.EventMetrics, 1000)
	go ep.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
