// Copyright 2017-2020 The Cloudprober Authors.
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

// Package sysvars implements a system variables exporter. It exports variables defined
// through an environment variable, as well other system variables like process uptime.
package sysvars

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
)

var (
	sysVarsMu sync.RWMutex
	sysVars   map[string]string
	l         *logger.Logger
	startTime time.Time
)

var cloudMetadataFlag = flag.String("cloud_metadata", "auto", "Collect cloud metadata for [auto|gce|ec2|none]")

var cloudProviders = struct {
	auto, gce, ec2 string
}{
	auto: "auto",
	gce:  "gce",
	ec2:  "ec2",
}

func GetVar(k string) string {
	sysVarsMu.RLock()
	defer sysVarsMu.RUnlock()
	if sysVars == nil {
		l.Error("sysVars is nil. sysvars.GetVar was called before sysvars.Init.")
		return ""
	}
	if sysVars[k] == "" {
		l.Error("sysvar " + k + " is not set.")
		return ""
	}
	return sysVars[k]
}

// Vars returns a copy of the system variables map, if already initialized.
// Otherwise an empty map is returned.
func Vars() map[string]string {
	vars := make(map[string]string)
	// We should never have to wait for these locks as sysVars are
	// updated inside Init and Init should be called only once, in
	// the beginning.
	sysVarsMu.RLock()
	defer sysVarsMu.RUnlock()
	if sysVars == nil {
		// Return an empty map if sysVars is not initialized yet.
		return vars
	}
	for k, v := range sysVars {
		vars[k] = v
	}
	return vars
}

func parseEnvVars(envVarsName string) map[string]string {
	envVars := make(map[string]string)
	if os.Getenv(envVarsName) == "" {
		return envVars
	}
	l.Infof("%s: %s", envVarsName, os.Getenv(envVarsName))
	for _, v := range strings.Split(os.Getenv(envVarsName), ",") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			l.Warningf("Bad env var: %s, skipping", v)
			continue
		}
		envVars[kv[0]] = kv[1]
	}
	return envVars
}

// StartTime returns cloudprober's start time.
func StartTime() time.Time {
	return startTime
}

func providersToCheck(fv string) []string {
	if fv == "" || fv == "none" {
		return nil
	}
	// Update this list when we add new providers
	if fv == cloudProviders.auto {
		return []string{cloudProviders.gce, cloudProviders.ec2}
	}
	return []string{fv}
}

func initCloudMetadata(fv string) error {
	for _, provider := range providersToCheck(fv) {
		switch provider {
		case cloudProviders.gce:
			onGCE, err := gceVars(sysVars, l)
			// Once we know it's GCE, don't continue checking.
			if onGCE {
				return err
			}
		case cloudProviders.ec2:
			tryHard := fv == cloudProviders.ec2
			onEC2, err := ec2Vars(sysVars, tryHard, l)
			// Once we know it's EC2, don't continue checking.
			if onEC2 {
				return err
			}
		default:
			return fmt.Errorf("unknown cloud provider: %v", provider)
		}
	}
	return nil
}

// Init initializes the sysvars module's global data structure. Init makes sure
// to initialize only once, further calls are a no-op. If needed, userVars
// can be passed to Init to add custom variables to sysVars. This can be useful
// for tests which require sysvars that might not exist, or might have the wrong
// value.
func Init(ll *logger.Logger, userVars map[string]string) error {
	sysVarsMu.Lock()
	defer sysVarsMu.Unlock()
	if sysVars != nil {
		return nil
	}

	l = ll
	startTime = time.Now()
	sysVars = map[string]string{
		"version": state.Version(),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("sysvars.Init(): error getting local hostname: %v", err)
	}
	sysVars["hostname"] = hostname

	if err := initCloudMetadata(*cloudMetadataFlag); err != nil {
		if *cloudMetadataFlag == cloudProviders.auto {
			return fmt.Errorf("got errror while initializing cloud metadata: %v, set flag: --cloud_metadata=none to disable cloud metadata check", err)
		}
		return err
	}

	for k, v := range userVars {
		sysVars[k] = v
	}
	return nil
}

// Start exports system variables at the given interval. It overlays variables with
// variables passed through the envVarsName env variable.
func Start(ctx context.Context, dataChan chan *metrics.EventMetrics, interval time.Duration, envVarsName string) {
	vars := Vars()
	for k, v := range parseEnvVars(envVarsName) {
		vars[k] = v
	}
	// Add reset timestamp (Unix epoch corresponding to when Cloudprober was started)
	vars["start_timestamp"] = strconv.FormatInt(startTime.Unix(), 10)

	var varsKeys []string
	for k := range vars {
		varsKeys = append(varsKeys, k)
	}
	sort.Strings(varsKeys)

	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", "sysvars").
		AddLabel("probe", "sysvars")
	em.Kind = metrics.GAUGE
	for _, k := range varsKeys {
		em.AddMetric(k, metrics.NewString(vars[k]))
	}
	l.Info(em.String())

	for ts := range time.Tick(interval) {
		// Don't run another cycles if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Update timestamp and publish static variables.
		em.Timestamp = ts
		dataChan <- em.Clone()
		l.Debug(em.String())

		runtimeVars(dataChan, l)
	}
}
