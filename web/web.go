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
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/common/httputils"
	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/probes/alerting"
	"github.com/cloudprober/cloudprober/servers"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/web/resources"
)

//go:embed static/*
var content embed.FS

var htmlTmpl = string(`
<html>
<head>
  <link href="/static/cloudprober.css" rel="stylesheet">
</head>

<body>
%s
<br><br><br><br>
%s
</body>
</html>
`)

var runningConfigTmpl = template.Must(template.New("runningConfig").Parse(`
<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
`))

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

	err := runningConfigTmpl.Execute(&statusBuf, struct {
		ProbesStatus, ServersStatus, SurfacersStatus interface{}
	}{
		ProbesStatus:    execTmpl(probes.StatusTmpl, probeInfo),
		SurfacersStatus: execTmpl(surfacers.StatusTmpl, surfacerInfo),
		ServersStatus:   execTmpl(servers.StatusTmpl, serverInfo),
	})

	if err != nil {
		return fmt.Sprintf(htmlTmpl, err.Error())
	}

	return fmt.Sprintf(htmlTmpl, resources.Header(), statusBuf.String())
}

func alertsState() string {
	status, err := alerting.StatusHTML()
	if err != nil {
		return fmt.Sprintf(htmlTmpl, err.Error())
	}
	return fmt.Sprintf(htmlTmpl, resources.Header(), status)
}

// Init initializes cloudprober web interface handler.
func Init() error {
	srvMux := runconfig.DefaultHTTPServeMux()
	for _, url := range []string{"/config", "/config-running", "/static/"} {
		if httputils.IsHandled(srvMux, url) {
			return fmt.Errorf("url %s is already handled", url)
		}
	}

	srvMux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, cloudprober.GetRawConfig())
	})

	parsedConfig := cloudprober.GetParsedConfig()
	srvMux.HandleFunc("/config-parsed", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, cloudprober.GetParsedConfig())
	})

	var configRunning string
	if !strings.Contains(parsedConfig, "{{ secret:$") {
		configRunning = runningConfig()
	} else {
		configRunning = `
		<p>Config contains secrets. /config-running is not available.<br>
		Visit <a href=/config-parsed>/config-parsed</a> to see the config.<p>
		`
	}
	srvMux.HandleFunc("/config-running", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, configRunning)
	})

	srvMux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, alertsState())
	})
	srvMux.Handle("/static/", http.FileServer(http.FS(content)))
	return nil
}
