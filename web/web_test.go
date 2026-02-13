// Copyright 2018-2026 The Cloudprober Authors.
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

package web

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/web/resources"
	"github.com/stretchr/testify/assert"
)

func TestInitWithDataFuncs(t *testing.T) {
	getInfo := func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {

		return nil, nil, nil
	}

	tests := []struct {
		name              string
		dataFuncs         DataFuncs
		withSecret        bool
		wantResp          map[string]string
		wantConfigRunning string
		wantErr           bool
	}{
		{
			name: "base",
			dataFuncs: DataFuncs{
				GetRawConfig:    func() string { return "raw-config" },
				GetParsedConfig: func() string { return "parsed-config" },
				GetInfo:         getInfo,
			},
			wantResp: map[string]string{
				"/config":        "raw-config",
				"/config-parsed": "parsed-config",
			},
		},
		{
			name: "with-secret",
			dataFuncs: DataFuncs{
				GetRawConfig:    func() string { return "raw-config" },
				GetParsedConfig: func() string { return "parsed-config **$password**" },
				GetInfo:         getInfo,
			},
			withSecret: true,
			wantResp: map[string]string{
				"/config":        "raw-config",
				"/config-parsed": "parsed-config **$password**",
			},
			wantConfigRunning: secretConfigRunningMsg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSrvMux := state.DefaultHTTPServeMux()
			defer state.SetDefaultHTTPServeMux(oldSrvMux)
			srvMux := http.NewServeMux()
			state.SetDefaultHTTPServeMux(srvMux)

			httpSrv := httptest.NewServer(srvMux)
			defer httpSrv.Close()

			if err := InitWithDataFuncs(tt.dataFuncs); (err != nil) != tt.wantErr {
				t.Errorf("InitWithDataFuncs() error = %v, wantErr %v", err, tt.wantErr)
			}

			client := httpSrv.Client()

			// Figure out config-running response
			var buf bytes.Buffer
			if !tt.withSecret {
				runningConfig(&buf, tt.dataFuncs)
			} else {
				buf.WriteString(resources.RenderPage(urlMap.RunningConfig, secretConfigRunningMsg))
			}
			tt.wantResp[urlMap.RunningConfig] = buf.String()

			for path, wantResp := range tt.wantResp {
				resp, err := client.Get(httpSrv.URL + path)
				if err != nil {
					t.Errorf("Error getting %s: %v", path, err)
					continue
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Got status code: %d, want: %d", resp.StatusCode, http.StatusOK)
				}
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Error reading response body: %v", err)
				}
				assert.Equal(t, wantResp, string(body), "response body")
			}

			httpSrv.Close()
		})
	}
}

func TestInit(t *testing.T) {
	assert.Nil(t, Init(), "Init() should return nil")
}

func TestAllLinksPageLinks(t *testing.T) {
	tests := []struct {
		name  string
		links []string
		want  []string
	}{
		{
			name:  "no static links",
			links: []string{"/link1", "/link2", "/link3"},
			want:  []string{"link1", "link2", "link3"},
		},
		{
			name:  "mixed links",
			links: []string{"/link1", "/static/link2", "/link3", "/static/link4", "/link5"},
			want:  []string{"link1", "link3", "link5"},
		},
		{
			name:  "only static links",
			links: []string{"/static/link1", "/static/link2"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, allLinksPageLinks(tt.links))
		})
	}
}

func TestArtifactsLinksWithManualRegistration(t *testing.T) {
	oldMux := state.DefaultHTTPServeMux()
	defer state.SetDefaultHTTPServeMux(oldMux)
	state.SetDefaultHTTPServeMux(http.NewServeMux())

	state.AddWebHandler("/manual-artifact", func(w http.ResponseWriter, r *http.Request) {}, state.WithArtifactsLink(""))
	state.AddWebHandler("/manual-artifact-custom", func(w http.ResponseWriter, r *http.Request) {}, state.WithArtifactsLink("/custom/artifact"))

	want := []string{"custom/artifact", "manual-artifact"}
	assert.Equal(t, want, artifactsLinks())
}

func verifyHeader(t *testing.T, path string, got string) {
	t.Helper()
	wantHeader := strings.TrimRight(resources.RenderPage(path, ""), "</body>\n</html>\n")
	assert.Contains(t, got, wantHeader)
}

func TestRunningConfig(t *testing.T) {
	tests := []struct {
		name    string
		getInfo func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo)
		wantAll []string
	}{
		{
			name: "nil info",
			getInfo: func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
				return nil, nil, nil
			},
			wantAll: []string{"<h3>Probes:</h3>", "<h3>Surfacers:</h3>", "<h3>Servers:</h3>"},
		},
		{
			name: "with probe info",
			getInfo: func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
				probeInfoMap := map[string]*probes.ProbeInfo{
					"test-probe": {
						Name:     "test-probe",
						Type:     "HTTP",
						Interval: "5s",
						Timeout:  "1s",
					},
				}
				return probeInfoMap, nil, nil
			},
			wantAll: []string{"<h3>Probes:</h3>", "test-probe", "<h3>Surfacers:</h3>", "<h3>Servers:</h3>"},
		},
		{
			name: "with surfacer info",
			getInfo: func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
				return nil, []*surfacers.SurfacerInfo{
					{Type: "PROMETHEUS", Name: "prom-surfacer"},
				}, nil
			},
			wantAll: []string{"<h3>Probes:</h3>", "<h3>Surfacers:</h3>", "prom-surfacer", "<h3>Servers:</h3>"},
		},
		{
			name: "with server info",
			getInfo: func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
				return nil, nil, []*servers.ServerInfo{
					{Type: "HTTP"},
				}
			},
			wantAll: []string{"<h3>Probes:</h3>", "<h3>Surfacers:</h3>", "<h3>Servers:</h3>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			fn := DataFuncs{
				GetRawConfig:    func() string { return "" },
				GetParsedConfig: func() string { return "" },
				GetInfo:         tt.getInfo,
			}
			runningConfig(&buf, fn)
			got := buf.String()
			for _, want := range tt.wantAll {
				assert.Contains(t, got, want)
			}
			verifyHeader(t, urlMap.RunningConfig, got)
		})
	}
}

func TestEndpoints(t *testing.T) {
	tests := []struct {
		url          string
		wantContains string
	}{
		{
			url:          "/links",
			wantContains: "All Links",
		},
		{
			url:          "/artifacts",
			wantContains: "global-artifacts/",
		},
		{
			url:          "/alerts",
			wantContains: "Alerts",
		},
		{
			url:          "/static/cloudprober.css",
			wantContains: "background-color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			oldSrvMux := state.DefaultHTTPServeMux()
			defer state.SetDefaultHTTPServeMux(oldSrvMux)
			srvMux := http.NewServeMux()
			state.SetDefaultHTTPServeMux(srvMux)

			state.AddWebHandler("/global-artifacts/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Global Artifacts")
			}, state.WithArtifactsLink("/global-artifacts/"))

			fn := DataFuncs{
				GetRawConfig:    func() string { return "" },
				GetParsedConfig: func() string { return "" },
				GetInfo: func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
					return nil, nil, nil
				},
			}

			err := InitWithDataFuncs(fn)
			assert.NoError(t, err)

			httpSrv := httptest.NewServer(srvMux)
			defer httpSrv.Close()

			resp, err := httpSrv.Client().Get(httpSrv.URL + tt.url)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			if tt.url != "/static/cloudprober.css" {
				verifyHeader(t, tt.url, string(body))
			}
			assert.Contains(t, string(body), tt.wantContains)
		})
	}
}
