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

	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/common/httputils"
	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/servers"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/web/resources"
)

//go:embed static/*
var content embed.FS

var runningConfigTmpl = `
<html>

<head>
  <link href="/static/cloudprober.css" rel="stylesheet">
</head>

<body>
{{.Header}}
<br><br><br><br>

<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
</body>
</html>
`

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

	tmpl, _ := template.New("runningConfig").Parse(runningConfigTmpl)
	tmpl.Execute(&statusBuf, struct {
		Header, ProbesStatus, ServersStatus, SurfacersStatus interface{}
	}{
		Header:          resources.Header(),
		ProbesStatus:    execTmpl(probes.StatusTmpl, probeInfo),
		SurfacersStatus: execTmpl(surfacers.StatusTmpl, surfacerInfo),
		ServersStatus:   execTmpl(servers.StatusTmpl, serverInfo),
	})

	return statusBuf.String()
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
		fmt.Fprint(w, cloudprober.GetTextConfig())
	})
	srvMux.HandleFunc("/config-running", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, runningConfig())
	})
	srvMux.Handle("/static/", http.FileServer(http.FS(content)))
	return nil
}
