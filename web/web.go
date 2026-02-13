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
	"io"
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

var runningConfigTmpl = template.Must(template.New("runningConfig").Parse(`
<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
`))

var urlMap = struct {
	Alerts        string
	RunningConfig string
	ParsedConfig  string
	RawConfig     string
	Links         string
	Artifacts     string
}{
	Alerts:        "/alerts",
	RunningConfig: "/config-running",
	ParsedConfig:  "/config-parsed",
	RawConfig:     "/config",
	Links:         "/links",
	Artifacts:     "/artifacts",
}

// runningConfig returns cloudprober's running config.
func runningConfig(w io.Writer, fn DataFuncs) {
	var statusBuf bytes.Buffer

	probeInfo, surfacerInfo, serverInfo := fn.GetInfo()

	err := runningConfigTmpl.Execute(&statusBuf, struct {
		ProbesStatus, ServersStatus, SurfacersStatus interface{}
	}{
		ProbesStatus:    resources.ExecTmpl(probes.StatusTmpl, probeInfo),
		SurfacersStatus: resources.ExecTmpl(surfacers.StatusTmpl, surfacerInfo),
		ServersStatus:   resources.ExecTmpl(servers.StatusTmpl, serverInfo),
	})

	if err != nil {
		w.Write([]byte(resources.RenderPage(urlMap.RunningConfig, err.Error())))
		return
	}

	w.Write([]byte(resources.RenderPage(urlMap.RunningConfig, statusBuf.String())))
}

func allLinksPageLinks(links []string) []string {
	var out []string
	for _, link := range links {
		if strings.Contains(link, "/static/") {
			continue
		}
		out = append(out, strings.TrimLeft(link, "/"))
	}
	sort.Strings(out)
	return out
}

func artifactsLinks() []string {
	var out []string
	for _, link := range state.ArtifactsURLs() {
		out = append(out, strings.TrimLeft(link, "/"))
	}
	sort.Strings(out)
	return out
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
	if err := state.AddWebHandler(urlMap.RawConfig, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetRawConfig())
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler(urlMap.ParsedConfig, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fn.GetParsedConfig())
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler(urlMap.RunningConfig, func(w http.ResponseWriter, r *http.Request) {
		parsedConfig := fn.GetParsedConfig()
		if !config.EnvRegex.MatchString(parsedConfig) {
			runningConfig(w, fn)
			return
		} else {
			w.Write([]byte(resources.RenderPage(urlMap.RunningConfig, secretConfigRunningMsg)))
		}
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler(urlMap.Alerts, func(w http.ResponseWriter, r *http.Request) {
		status, err := alerting.StatusHTML()
		if err != nil {
			w.Write([]byte(resources.RenderPage(urlMap.Alerts, err.Error())))
			return
		}
		w.Write([]byte(resources.RenderPage(urlMap.Alerts, status)))
	}); err != nil {
		return err
	}

	if err := state.AddWebHandler(urlMap.Links, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(resources.LinksPage(urlMap.Links, "All Links", allLinksPageLinks(state.AllLinks()))))
	}); err != nil {
		return err
	}

	artifactsL := artifactsLinks()
	if len(artifactsL) > 0 {
		if err := state.AddWebHandler(urlMap.Artifacts, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(resources.LinksPage(urlMap.Artifacts, "Artifacts", artifactsLinks())))
		}); err != nil {
			return err
		}
	}

	return state.AddWebHandler("/static/", http.FileServer(http.FS(content)).ServeHTTP)
}
