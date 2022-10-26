// Copyright 2018 The Cloudprober Authors.
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

// Package web provides web interface for cloudprober.
package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/servers"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/sysvars"
)

func execTmpl(tmpl *template.Template, v interface{}) template.HTML {
	var statusBuf bytes.Buffer
	err := tmpl.Execute(&statusBuf, v)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(err.Error()))
	}
	return template.HTML(statusBuf.String())
}

// runningConfig returns cloudprober's running config.
func runningConfig() string {
	var statusBuf bytes.Buffer

	probeInfo, surfacerInfo, serverInfo := cloudprober.GetInfo()
	startTime := sysvars.StartTime().Truncate(time.Millisecond)
	uptime := time.Since(startTime)

	tmpl, _ := template.New("runningConfig").Parse(runningConfigTmpl)
	tmpl.Execute(&statusBuf, struct {
		Version, StartTime, Uptime, ProbesStatus, ServersStatus, SurfacersStatus interface{}
	}{
		Version:         runconfig.Version(),
		StartTime:       startTime,
		Uptime:          uptime.String(),
		ProbesStatus:    execTmpl(probes.StatusTmpl, probeInfo),
		SurfacersStatus: execTmpl(surfacers.StatusTmpl, surfacerInfo),
		ServersStatus:   execTmpl(servers.StatusTmpl, serverInfo),
	})

	return statusBuf.String()
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, cloudprober.GetTextConfig())
}

func runningConfigHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, runningConfig())
}

// Init initializes cloudprober web interface handler.
func Init() {
	srvMux := runconfig.DefaultHTTPServeMux()
	srvMux.HandleFunc("/config", configHandler)
	srvMux.HandleFunc("/config-running", runningConfigHandler)
}
