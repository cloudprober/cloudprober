// Copyright 2018-2024 The Cloudprober Authors.
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

	"github.com/cloudprober/cloudprober/config"
	"github.com/cloudprober/cloudprober/internal/alerting"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/web/resources"
	"github.com/cloudprober/cloudprober/web/webutils"
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
func runningConfig(fn DataFuncs) string {
	var statusBuf bytes.Buffer

	probeInfo, surfacerInfo, serverInfo := fn.GetInfo()

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

type DataFuncs struct {
	GetRawConfig    func() string
	GetParsedConfig func() string
	GetInfo         func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo)
}

func Init() error {
	l := logger.Logger{}
	l.Warningf("web.Init is a no-op now. Web interface is now initialized by cloudprober.Init(), you don't need to initialize it explicitly.")
	return nil
}

var secretConfigRunningMsg = `
	<p>Config contains secrets. /config-running is not available.<br>
	Visit <a href=/config-parsed>/config-parsed</a> to see the config.<p>
	`

// InitWithDataFuncs initializes cloudprober web interface handler.
func InitWithDataFuncs(fn DataFuncs) error {
	srvMux := state.DefaultHTTPServeMux()
	for _, url := range []string{"/config", "/config-running", "/static/"} {
		if webutils.IsHandled(srvMux, url) {
			return fmt.Errorf("url %s is already handled", url)
		}
	}

	srvMux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetRawConfig())
	})

	srvMux.HandleFunc("/config-parsed", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetParsedConfig())
	})

	srvMux.HandleFunc("/config-running", func(w http.ResponseWriter, r *http.Request) {
		parsedConfig := fn.GetParsedConfig()
		var configRunning string
		if !config.EnvRegex.MatchString(parsedConfig) {
			configRunning = runningConfig(fn)
		} else {
			configRunning = secretConfigRunningMsg
		}

		fmt.Fprint(w, configRunning)
	})

	srvMux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, alertsState())
	})
	srvMux.Handle("/static/", http.FileServer(http.FS(content)))
	return nil
}
