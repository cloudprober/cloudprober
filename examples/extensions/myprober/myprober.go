package main

import (
	"context"
	"flag"

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/examples/extensions/myprober/myprobe"
	"github.com/cloudprober/cloudprober/examples/extensions/myprober/mytargets"
	"github.com/cloudprober/cloudprober/logger"
)

func main() {
	flag.Parse()

	var log = logger.New()

	myprobe.Init()
	mytargets.Init()

	if err := cloudprober.Init(); err != nil {
		log.Criticalf("Error initializing cloudprober. Err: %v", err)
	}

	cloudprober.Start(context.Background())

	// Wait forever
	select {}
}
