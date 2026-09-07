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

// Package resources provides webpages related resources.
package resources

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/state"
)

type headerTmplData struct {
	Version, VersionTitle, Uptime, UptimeTitle, StatusLink, LinksPrefix string
	IncludeMetricsLink, IncludeArtifactsLink                            bool
}

var t = template.Must(template.New("header").Parse(`
<header>
  <a href="https://cloudprober.org"><img class="logo" src="{{.LinksPrefix}}static/cloudprober-horizontal.svg" alt="Cloudprober" width="170" height="60"></a>
  {{if .Version}}<div class="version"{{if .VersionTitle}} title="{{.VersionTitle}}"{{end}}>{{.Version}}</div>{{end}}
</header> 
<hr/>
<div style="float:left">
  <div class="uptime" title="{{.UptimeTitle}}"><b>Uptime</b>: {{.Uptime}}</div>
  <b>Links</b> (<a href="{{.LinksPrefix}}links">all</a>):
  	<a href="{{.LinksPrefix}}{{.StatusLink}}">/status</a>,
	<a href="{{.LinksPrefix}}config-running">/config</a> (<a href="{{.LinksPrefix}}config-parsed">parsed</a> | <a href="{{.LinksPrefix}}config">raw</a>),
	<a href="{{.LinksPrefix}}logs">/logs</a>,
	{{if .IncludeMetricsLink -}} <a href="{{.LinksPrefix}}metrics">/metrics</a>,{{ end }}
	{{if .IncludeArtifactsLink -}} <a href="{{.LinksPrefix}}artifacts">/artifacts</a>,{{ end }}
	<a href="{{.LinksPrefix}}alerts">/alerts</a>
</div>
`))

func headerData(linksPrefix string) headerTmplData {
	startTime := sysvars.StartTime().Truncate(time.Millisecond)
	uptime := time.Since(startTime).Truncate(time.Millisecond)

	// version and buildTimestamp are set through -ldflags, so a plain "go
	// build" binary has neither. Drop the tag and its tooltip in that case
	// instead of rendering an empty version or a "built at" of the zero time.
	versionTitle := ""
	if builtAt := state.BuildTimestamp(); !builtAt.IsZero() {
		versionTitle = "Built at " + builtAt.String()
	}

	includeMetrics := false
	includeArtifacts := false
	statusLink := "status"
	allLinks := state.AllLinks()
	for _, link := range allLinks {
		if strings.Contains(link, "/artifacts") {
			includeArtifacts = true
		}
		if link == "/metrics" {
			includeMetrics = true
		}
		if strings.HasSuffix(link, "/status") {
			statusLink = strings.TrimLeft(link, "/")
		}
	}

	return headerTmplData{
		Version:              state.Version(),
		VersionTitle:         versionTitle,
		Uptime:               humanizeDuration(uptime),
		UptimeTitle:          "Started " + startTime.String(),
		StatusLink:           statusLink,
		LinksPrefix:          linksPrefix,
		IncludeMetricsLink:   includeMetrics,
		IncludeArtifactsLink: includeArtifacts,
	}
}

// humanizeDuration formats d using its two most significant units, e.g. "3d 4h"
// or "12m 30s", dropping the smaller one when it is zero. Go's
// Duration.String() stops at hours, which turns a long uptime into something
// like "2562047h47m16.854s". Days are deliberately the largest unit: a "year"
// is 365 or 365.25 days depending on who is reading, while "423d" is
// unambiguous and still scans.
func humanizeDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}

	var major, minor int
	var majorUnit, minorUnit string
	switch {
	case d >= 24*time.Hour:
		major, majorUnit = int(d/(24*time.Hour)), "d"
		minor, minorUnit = int(d/time.Hour)%24, "h"
	case d >= time.Hour:
		major, majorUnit = int(d/time.Hour), "h"
		minor, minorUnit = int(d/time.Minute)%60, "m"
	case d >= time.Minute:
		major, majorUnit = int(d/time.Minute), "m"
		minor, minorUnit = int(d/time.Second)%60, "s"
	default:
		return fmt.Sprintf("%ds", int(d/time.Second))
	}

	if minor == 0 {
		return fmt.Sprintf("%d%s", major, majorUnit)
	}
	return fmt.Sprintf("%d%s %d%s", major, majorUnit, minor, minorUnit)
}

func Header(linksPrefix string) template.HTML {
	var buf bytes.Buffer
	if err := t.Execute(&buf, headerData(linksPrefix)); err != nil {
		panic(fmt.Sprintf("Error rendering header: %v", err))
	}
	return template.HTML(buf.String())
}

func LinkPrefixFromCurrentPath(path string) string {
	pathParts := strings.Split(path, "/")
	linkPrefix := ""
	for i := 2; i < len(pathParts); i++ {
		linkPrefix += "../"
	}
	return linkPrefix
}
