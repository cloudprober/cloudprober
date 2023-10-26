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

// This program implements a stand-alone ping prober binary using the
// cloudprober/ping package. It's intended to help prototype the ping package
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
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/probes/ping"
	configpb "github.com/cloudprober/cloudprober/probes/ping/proto"
	"github.com/cloudprober/cloudprober/targets"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	targetsF = flag.String("targets", "www.google.com,www.yahoo.com", "Comma separated list of targets")
	config   = flag.String("config_file", "", "Config file to change ping probe options (see probes/ping/config.proto for default options)")
	ipVer    = flag.Int("ip", 0, "IP version")
)

func main() {
	flag.Parse()

	probeConfig := &configpb.ProbeConf{}
	if *config != "" {
		b, err := os.ReadFile(*config)
		if err != nil {
			log.Fatal(err)
		}
		if err = prototext.Unmarshal(b, probeConfig); err != nil {
			log.Fatalf("error while parsing config: %v", err)
		}
	}

	opts := &options.Options{
		Targets:     targets.StaticTargets(*targetsF),
		Interval:    2 * time.Second,
		Timeout:     time.Second,
		LatencyUnit: 1 * time.Millisecond,
		ProbeConf:   probeConfig,
		LogMetrics:  func(*metrics.EventMetrics) {},
		IPVersion:   *ipVer,
	}
	p := &ping.Probe{}
	if err := p.Init("ping", opts); err != nil {
		log.Fatalf("error initializing ping probe from config: %v", err)
	}

	dataChan := make(chan *metrics.EventMetrics, 1000)
	go p.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
