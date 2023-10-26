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

// This program implements a stand-alone TCP prober binary using the
// cloudprober/tcp package. It's intended to help prototype the tcp package
// quickly and doesn't provide the facilities that cloudprober provides.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"flag"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/probes/tcp"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets"
)

var (
	intervalF = flag.Duration("interval", time.Second*30, "Interval between probes")
	timeoutF  = flag.Duration("timeout", time.Second*1, "Per-probe timeout")
	targetsF  = flag.String("targets", "www.google.com:80", "Static host targets.")
)

func main() {
	flag.Parse()

	c := &configpb.ProbeConf{}

	opts := options.DefaultOptions()
	opts.Interval, opts.Timeout = *intervalF, *timeoutF
	opts.Targets = targets.StaticTargets(*targetsF)
	opts.ProbeConf = c

	tp := &tcp.Probe{}
	if err := tp.Init("tcp_test", opts); err != nil {
		log.Fatalf("Error in initializing probe %s from the config. Err: %v", "http_test", err)
	}
	dataChan := make(chan *metrics.EventMetrics, 1000)
	go tp.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
