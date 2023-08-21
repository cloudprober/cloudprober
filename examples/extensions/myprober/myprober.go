package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/config"
	"github.com/cloudprober/cloudprober/examples/extensions/myprober/myprobe"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/web"
	"github.com/golang/glog"
)

var (
	configFile = flag.String("config_file", "", "Config file")
)

const (
	configMetadataKeyName = "cloudprober_config"
	defaultConfigFile     = "/etc/cloudprober.cfg"
)

func configFileToString(fileName string) string {
	b, err := ioutil.ReadFile(fileName)
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

	// Register stubby probe type
	probes.RegisterProbeType(int(myprobe.E_RedisProbe.Field),
		func() probes.Probe { return &myprobe.Probe{} })

	if err := cloudprober.InitFromConfig(""); err != nil {
		glog.Exitf("Error initializing cloudprober. Err: %v", err)
	}

	// web.Init sets up web UI for cloudprober.
	if err := web.Init(); err != nil {
		glog.Exitf("Error initializing web interface. Err: %v", err)
	}

	cloudprober.Start(context.Background())

	// Wait forever
	select {}
}
