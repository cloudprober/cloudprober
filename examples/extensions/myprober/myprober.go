package main

import (
	"context"
	"flag"

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/examples/extensions/myprober/myprobe"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/web"
)

func main() {
	flag.Parse()

	var log = logger.New()

	// Register stubby probe type
	probes.RegisterProbeType(int(myprobe.E_RedisProbe.TypeDescriptor().Number()),
		func() probes.Probe { return &myprobe.Probe{} })

	if err := cloudprober.Init(); err != nil {
		log.Criticalf("Error initializing cloudprober. Err: %v", err)
	}

	// web.Init sets up web UI for cloudprober.
	if err := web.Init(); err != nil {
		log.Criticalf("Error initializing web interface. Err: %v", err)
	}

	cloudprober.Start(context.Background())

	// Wait forever
	select {}
}
