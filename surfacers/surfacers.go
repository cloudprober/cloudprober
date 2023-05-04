// Copyright 2017-2021 The Cloudprober Authors.
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
Package surfacers is the base package for creating Surfacer objects that are
used for writing metics data to different monitoring services.

Any Surfacer that is created for writing metrics data to a monitor system
should implement the below Surfacer interface and should accept
metrics.EventMetrics object through a Write() call. Each new surfacer should
also plug itself in through the New() method defined here.
*/
package surfacers

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/bigquery"
	"github.com/cloudprober/cloudprober/surfacers/cloudwatch"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	"github.com/cloudprober/cloudprober/surfacers/common/transform"
	"github.com/cloudprober/cloudprober/surfacers/datadog"
	"github.com/cloudprober/cloudprober/surfacers/file"
	"github.com/cloudprober/cloudprober/surfacers/postgres"
	"github.com/cloudprober/cloudprober/surfacers/probestatus"
	"github.com/cloudprober/cloudprober/surfacers/prometheus"
	"github.com/cloudprober/cloudprober/surfacers/pubsub"
	"github.com/cloudprober/cloudprober/surfacers/stackdriver"
	"github.com/cloudprober/cloudprober/web/formatutils"

	surfacerpb "github.com/cloudprober/cloudprober/surfacers/proto"
)

var (
	userDefinedSurfacers   = make(map[string]Surfacer)
	userDefinedSurfacersMu sync.Mutex
)

// StatusTmpl variable stores the HTML template suitable to generate the
// surfacers' status for cloudprober's /status page. It expects an array of
// SurfacerInfo objects as input.
var StatusTmpl = template.Must(template.New("statusTmpl").Parse(`
<table class="status-list">
  <tr>
    <th>Type</th>
    <th>Name</th>
    <th>Conf</th>
  </tr>
  {{ range . }}
  <tr>
    <td>{{.Type}}</td>
    <td>{{.Name}}</td>
    <td>
    {{if .Conf}}
      <pre>{{.Conf}}</pre>
    {{else}}
      default
    {{end}}
    </td>
  </tr>
  {{ end }}
</table>
`))

// Default surfacers. These surfacers are enabled if no surfacer is defined.
var defaultSurfacers = []*surfacerpb.SurfacerDef{
	{
		Type: surfacerpb.Type_PROMETHEUS.Enum(),
	},
	{
		Type: surfacerpb.Type_FILE.Enum(),
	},
}

// Required surfacers. These surfacers are enabled by default unless explicitly
// disabled in their own configs.
var requiredSurfacers = []*surfacerpb.SurfacerDef{
	{
		Type: surfacerpb.Type_PROBESTATUS.Enum(),
	},
}

// Surfacer is an interface for all metrics surfacing systems
type Surfacer interface {
	// Function for writing a piece of metric data to a specified metric
	// store (or other location).
	Write(ctx context.Context, em *metrics.EventMetrics)
}

type surfacerWrapper struct {
	Surfacer
	opts    *options.Options
	lvCache map[string]*metrics.EventMetrics
}

func (sw *surfacerWrapper) Write(ctx context.Context, em *metrics.EventMetrics) {
	if !sw.opts.AllowEventMetrics(em) {
		return
	}

	if sw.opts.AddFailureMetric {
		if err := transform.AddFailureMetric(em); err != nil {
			sw.opts.Logger.Warning(err.Error())
		}
	}

	if sw.opts.Config.GetExportAsGauge() && em.Kind == metrics.CUMULATIVE {
		newEM, err := transform.CumulativeToGauge(em, sw.lvCache, sw.opts.Logger)
		if err != nil {
			sw.opts.Logger.Errorf("Error converting CUMULATIVE metrics to GAUGE: %v", err)
			return
		}
		em = newEM
	}

	sw.Surfacer.Write(ctx, em)
}

// SurfacerInfo encapsulates a Surfacer and related info.
type SurfacerInfo struct {
	Surfacer
	Type string
	Name string
	Conf string
}

func inferType(s *surfacerpb.SurfacerDef) surfacerpb.Type {
	switch s.Surfacer.(type) {
	case *surfacerpb.SurfacerDef_PrometheusSurfacer:
		return surfacerpb.Type_PROMETHEUS
	case *surfacerpb.SurfacerDef_StackdriverSurfacer:
		return surfacerpb.Type_STACKDRIVER
	case *surfacerpb.SurfacerDef_FileSurfacer:
		return surfacerpb.Type_FILE
	case *surfacerpb.SurfacerDef_PostgresSurfacer:
		return surfacerpb.Type_POSTGRES
	case *surfacerpb.SurfacerDef_PubsubSurfacer:
		return surfacerpb.Type_PUBSUB
	case *surfacerpb.SurfacerDef_CloudwatchSurfacer:
		return surfacerpb.Type_CLOUDWATCH
	case *surfacerpb.SurfacerDef_DatadogSurfacer:
		return surfacerpb.Type_DATADOG
	case *surfacerpb.SurfacerDef_ProbestatusSurfacer:
		return surfacerpb.Type_PROBESTATUS
	case *surfacerpb.SurfacerDef_BigquerySurfacer:
		return surfacerpb.Type_BIGQUERY

	}

	return surfacerpb.Type_NONE
}

// initSurfacer initializes and returns a new surfacer based on the config.
func initSurfacer(ctx context.Context, s *surfacerpb.SurfacerDef, sType surfacerpb.Type) (Surfacer, interface{}, error) {
	// Create a new logger
	logName := s.GetName()
	if logName == "" {
		logName = strings.ToLower(s.GetType().String())
	}

	l, err := logger.NewCloudproberLog(logName)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create cloud logger: %v", err)
	}

	opts, err := options.BuildOptionsFromConfig(s, l)
	if err != nil {
		return nil, nil, err
	}

	var conf interface{}
	var surfacer Surfacer

	switch sType {
	case surfacerpb.Type_PROMETHEUS:
		surfacer, err = prometheus.New(ctx, s.GetPrometheusSurfacer(), opts, l)
		conf = s.GetPrometheusSurfacer()
	case surfacerpb.Type_STACKDRIVER:
		surfacer, err = stackdriver.New(ctx, s.GetStackdriverSurfacer(), opts, l)
		conf = s.GetStackdriverSurfacer()
	case surfacerpb.Type_FILE:
		surfacer, err = file.New(ctx, s.GetFileSurfacer(), opts, l)
		conf = s.GetFileSurfacer()
	case surfacerpb.Type_POSTGRES:
		surfacer, err = postgres.New(ctx, s.GetPostgresSurfacer(), l)
		conf = s.GetPostgresSurfacer()
	case surfacerpb.Type_PUBSUB:
		surfacer, err = pubsub.New(ctx, s.GetPubsubSurfacer(), opts, l)
		conf = s.GetPubsubSurfacer()
	case surfacerpb.Type_CLOUDWATCH:
		surfacer, err = cloudwatch.New(ctx, s.GetCloudwatchSurfacer(), opts, l)
		conf = s.GetCloudwatchSurfacer()
	case surfacerpb.Type_DATADOG:
		surfacer, err = datadog.New(ctx, s.GetDatadogSurfacer(), opts, l)
		conf = s.GetDatadogSurfacer()
	case surfacerpb.Type_PROBESTATUS:
		surfacer, err = probestatus.New(ctx, s.GetProbestatusSurfacer(), opts, l)
		conf = s.GetProbestatusSurfacer()
	case surfacerpb.Type_BIGQUERY:
		surfacer, err = bigquery.New(ctx, s.GetBigquerySurfacer(), opts, l)
		conf = s.GetBigquerySurfacer()
	case surfacerpb.Type_USER_DEFINED:
		userDefinedSurfacersMu.Lock()
		defer userDefinedSurfacersMu.Unlock()
		surfacer = userDefinedSurfacers[s.GetName()]
		if surfacer == nil {
			return nil, nil, fmt.Errorf("unregistered user defined surfacer: %s", s.GetName())
		}
	default:
		return nil, nil, fmt.Errorf("unknown surfacer type: %s", s.GetType())
	}

	return &surfacerWrapper{
		Surfacer: surfacer,
		opts:     opts,
		lvCache:  make(map[string]*metrics.EventMetrics),
	}, conf, err
}

// Init initializes the surfacers from the config protobufs and returns them as
// a list.
func Init(ctx context.Context, sDefs []*surfacerpb.SurfacerDef) ([]*SurfacerInfo, error) {
	// If no surfacers are defined, return default surfacers. This behavior
	// can be disabled by explicitly specifying "surfacer {}" in the config.
	if len(sDefs) == 0 {
		sDefs = defaultSurfacers
	}

	foundSurfacers := make(map[surfacerpb.Type]bool)

	var result []*SurfacerInfo
	for _, sDef := range sDefs {
		sType := sDef.GetType()

		if sType == surfacerpb.Type_NONE {
			// Don't do anything if surfacer type is NONE and nothing is defined inside
			// it: for example: "surfacer{}". This is one of the ways to disable
			// surfacers as not adding surfacers at all results in default surfacers
			// being added automatically.
			if sDef.Surfacer == nil {
				continue
			}
			sType = inferType(sDef)
		}

		s, conf, err := initSurfacer(ctx, sDef, sType)
		if err != nil {
			return nil, err
		}

		foundSurfacers[sType] = true

		result = append(result, &SurfacerInfo{
			Surfacer: s,
			Type:     sType.String(),
			Name:     sDef.GetName(),
			Conf:     formatutils.ConfToString(conf),
		})
	}

	for _, s := range requiredSurfacers {
		if !foundSurfacers[s.GetType()] {
			surfacer, _, err := initSurfacer(ctx, s, s.GetType())
			if err != nil {
				return nil, err
			}
			result = append(result, &SurfacerInfo{
				Surfacer: surfacer,
				Type:     s.GetType().String(),
			})
		}
	}
	return result, nil
}

// Register allows you to register a user defined surfacer with cloudprober.
// Example usage:
//
//	import (
//		"github.com/cloudprober/cloudprober"
//		"github.com/cloudprober/cloudprober/surfacers"
//	)
//
//	s := &FancySurfacer{}
//	surfacers.Register("fancy_surfacer", s)
//	pr, err := cloudprober.InitFromConfig(*configFile)
//	if err != nil {
//		log.Exitf("Error initializing cloudprober. Err: %v", err)
//	}
func Register(name string, s Surfacer) {
	userDefinedSurfacersMu.Lock()
	defer userDefinedSurfacersMu.Unlock()
	userDefinedSurfacers[name] = s
}
