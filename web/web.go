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
	"sort"
	"strings"

	"github.com/cloudprober/cloudprober/config"
	"github.com/cloudprober/cloudprober/internal/alerting"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/state"
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

var allLinksTmpl = template.Must(template.New("allLinks").Parse(`
<html>
<h3>Links:</h3>
<ul>
  {{ range .}}
  <li><a href="{{.}}">{{.}}</a></li>
  {{ end }}
</ul>
</html>
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

func allLinksPage() string {
	links := state.AllLinks()
	var out []string
	for _, link := range links {
		if strings.Contains(link, "/static/") {
			continue
		}
		out = append(out, link)
	}
	sort.Strings(out)
	return fmt.Sprintf(htmlTmpl, resources.Header(), execTmpl(allLinksTmpl, out))
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
	if err := state.AddWebHandler("/config", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetRawConfig())
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler("/config-parsed", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetParsedConfig())
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler("/config-running", func(w http.ResponseWriter, r *http.Request) {
		parsedConfig := fn.GetParsedConfig()
		var configRunning string
		if !config.EnvRegex.MatchString(parsedConfig) {
			configRunning = runningConfig(fn)
		} else {
			configRunning = secretConfigRunningMsg
		}

		fmt.Fprint(w, configRunning)
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler("/alerts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, alertsState())
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler("/links", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, allLinksPage())
	}); err != nil {
		return err
	}

	return state.AddWebHandler("/static/", http.FileServer(http.FS(content)).ServeHTTP)
}
