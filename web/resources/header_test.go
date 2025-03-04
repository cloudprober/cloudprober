// Copyright 2025 The Cloudprober Authors.
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

package resources

import (
	"html/template"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/state"
	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	state.SetVersion("v1.0.0")
	state.SetBuildTimestamp(time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC))
	state.SetDefaultHTTPServeMux(http.NewServeMux())
	if err := state.AddWebHandler("/artifacts/", func(w http.ResponseWriter, r *http.Request) {}); err != nil {
		t.Fatal(err)
	}
	if err := state.AddWebHandler("/metrics", func(w http.ResponseWriter, r *http.Request) {}); err != nil {
		t.Fatal(err)
	}

	expected := `
<header>
  <a href="https://cloudprober.org">Cloudprober</a> (<a href="https://github.com/cloudprober/cloudprober">Github</a>)
</header> 
<hr/>
<div style="float:left">
  <b>Started</b>: 0001-01-01 00:00:00 &#43;0000 UTC -- up 2562047h47m16.854s<br/>
  <b>Version</b>: v1.0.0<br>
  <b>Built at</b>: 2023-10-01 12:00:00 &#43;0000 UTC<br>
  <b>Other Links </b>(<a href="links">all</a>):
  	<a href="status">/status</a>,
	<a href="config-running">/config</a> (<a href="config-parsed">parsed</a> | <a href="config">raw</a>),
	<a href="metrics">/metrics</a>,
	<a href="artifacts">/artifacts</a>,
	<a href="alerts">/alerts</a>
</div>
`

	t.Run("no prefix", func(t *testing.T) {
		assert.Equal(t, template.HTML(expected), Header(""))
	})

	t.Run("with prefix", func(t *testing.T) {
		expected = regexp.MustCompile(`href="([^".]*)"`).ReplaceAllString(expected, "href=\"../../$1\"")
		assert.Equal(t, template.HTML(expected), Header("../../"))
	})
}

func TestHeaderData(t *testing.T) {
	tests := []struct {
		name                string
		version             string
		buildTimestamp      time.Time
		links               []string
		wantStatusLink      string
		expectMetricsLink   bool
		expectArtifactsLink bool
	}{
		{
			name:           "No links",
			version:        "v1.0.0",
			buildTimestamp: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:          []string{},
		},
		{
			name:           "Status link",
			version:        "v1.0.0",
			buildTimestamp: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:          []string{"/my/probe/status"},
			wantStatusLink: "my/probe/status",
		},
		{
			name:                "Metrics link",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:               []string{"/metrics"},
			expectMetricsLink:   true,
			expectArtifactsLink: false,
		},
		{
			name:                "Artifacts link",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:               []string{"/artifacts"},
			expectMetricsLink:   false,
			expectArtifactsLink: true,
		},
		{
			name:                "Both links",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:               []string{"/metrics", "/artifacts"},
			expectMetricsLink:   true,
			expectArtifactsLink: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state.SetVersion(tt.version)
			state.SetBuildTimestamp(tt.buildTimestamp)

			state.SetDefaultHTTPServeMux(http.NewServeMux())
			for _, link := range tt.links {
				if err := state.AddWebHandler(link, func(w http.ResponseWriter, r *http.Request) {}); err != nil {
					t.Fatal(err)
				}
			}

			wantStartTime := sysvars.StartTime().Truncate(time.Millisecond)
			wantUptime := time.Since(wantStartTime).Truncate(time.Millisecond)
			if tt.wantStatusLink == "" {
				tt.wantStatusLink = "status"
			}

			for _, linksPrefix := range []string{"", "../"} {
				data := headerData(linksPrefix)
				assert.Equal(t, tt.version, data.Version)
				assert.Equal(t, tt.buildTimestamp, data.BuiltAt)
				assert.Equal(t, wantStartTime, data.StartTime)
				assert.Equal(t, wantUptime, data.Uptime)
				assert.Equal(t, tt.expectMetricsLink, data.IncludeMetricsLink)
				assert.Equal(t, tt.expectArtifactsLink, data.IncludeArtifactsLink)
				assert.Equal(t, linksPrefix, data.LinksPrefix)
				assert.Equal(t, tt.wantStatusLink, data.StatusLink)
			}
		})
	}
}

func TestLinkPrefixFromCurrentPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{path: "/", expected: ""},
		{path: "/status", expected: ""},
		{path: "/config/running", expected: "../"},
		{path: "/some/deep/path", expected: "../../"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, LinkPrefixFromCurrentPath(tt.path))
		})
	}
}
