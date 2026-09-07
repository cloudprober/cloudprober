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
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/state"
	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	oldVersion := state.Version()
	oldBuiltAt := state.BuildTimestamp()
	oldMux := state.DefaultHTTPServeMux()
	defer state.SetVersion(oldVersion)
	defer state.SetBuildTimestamp(oldBuiltAt)
	defer state.SetDefaultHTTPServeMux(oldMux)

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
  <a href="https://cloudprober.org"><img class="logo" src="static/cloudprober-horizontal.svg" alt="Cloudprober" width="170" height="60"></a>
  <div class="version" title="Built at 2023-10-01 12:00:00 &#43;0000 UTC">v1.0.0</div>
</header> 
<hr/>
<div style="float:left">
  <div class="uptime" title="Started 0001-01-01 00:00:00 &#43;0000 UTC"><b>Uptime</b>: 106751d 23h</div>
  <b>Links</b> (<a href="links">all</a>):
  	<a href="status">/status</a>,
	<a href="config-running">/config</a> (<a href="config-parsed">parsed</a> | <a href="config">raw</a>),
	<a href="logs">/logs</a>,
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
		expected = strings.ReplaceAll(expected, `src="static/`, `src="../../static/`)
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
		wantVersionTitle    string
		expectMetricsLink   bool
		expectArtifactsLink bool
	}{
		{
			// No -ldflags: no version tag, and no "built at" tooltip on it.
			name:  "Unset version and build timestamp",
			links: []string{},
		},
		{
			name:             "No links",
			version:          "v1.0.0",
			buildTimestamp:   time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:            []string{},
			wantVersionTitle: "Built at 2023-10-01 12:00:00 +0000 UTC",
		},
		{
			name:             "Status link",
			version:          "v1.0.0",
			buildTimestamp:   time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			links:            []string{"/my/probe/status"},
			wantStatusLink:   "my/probe/status",
			wantVersionTitle: "Built at 2023-10-01 12:00:00 +0000 UTC",
		},
		{
			name:                "Metrics link",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			wantVersionTitle:    "Built at 2023-10-01 12:00:00 +0000 UTC",
			links:               []string{"/metrics"},
			expectMetricsLink:   true,
			expectArtifactsLink: false,
		},
		{
			name:                "Artifacts link",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			wantVersionTitle:    "Built at 2023-10-01 12:00:00 +0000 UTC",
			links:               []string{"/artifacts"},
			expectMetricsLink:   false,
			expectArtifactsLink: true,
		},
		{
			name:                "Both links",
			version:             "v1.0.0",
			buildTimestamp:      time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			wantVersionTitle:    "Built at 2023-10-01 12:00:00 +0000 UTC",
			links:               []string{"/metrics", "/artifacts"},
			expectMetricsLink:   true,
			expectArtifactsLink: true,
		},
	}

	oldVersion := state.Version()
	oldBuiltAt := state.BuildTimestamp()
	oldMux := state.DefaultHTTPServeMux()
	defer state.SetVersion(oldVersion)
	defer state.SetBuildTimestamp(oldBuiltAt)
	defer state.SetDefaultHTTPServeMux(oldMux)

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

			// sysvars.StartTime() is the zero time here, so time.Since()
			// saturates at the max duration and the uptime is a constant.
			wantStartTime := sysvars.StartTime().Truncate(time.Millisecond)
			wantUptime := humanizeDuration(time.Since(wantStartTime).Truncate(time.Millisecond))
			if tt.wantStatusLink == "" {
				tt.wantStatusLink = "status"
			}

			for _, linksPrefix := range []string{"", "../"} {
				data := headerData(linksPrefix)
				assert.Equal(t, tt.version, data.Version)
				assert.Equal(t, tt.wantVersionTitle, data.VersionTitle)
				assert.Equal(t, wantUptime, data.Uptime)
				assert.Equal(t, "Started "+wantStartTime.String(), data.UptimeTitle)
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

func TestHumanizeDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{d: 0, want: "0s"},
		{d: 900 * time.Millisecond, want: "0s"},
		{d: -5 * time.Second, want: "0s"}, // clock skew
		{d: 45 * time.Second, want: "45s"},
		{d: 12*time.Minute + 30*time.Second, want: "12m 30s"},
		{d: 12 * time.Minute, want: "12m"},
		{d: 4*time.Hour + 12*time.Minute, want: "4h 12m"},
		{d: 4 * time.Hour, want: "4h"},
		{d: 3*24*time.Hour + 4*time.Hour, want: "3d 4h"},
		{d: 3 * 24 * time.Hour, want: "3d"},
		// Sub-unit remainders are dropped, not rounded.
		{d: 3*24*time.Hour + 59*time.Minute, want: "3d"},
		// Days keep counting rather than rolling over into years.
		{d: 423*24*time.Hour + 4*time.Hour, want: "423d 4h"},
		// What time.Since() saturates to for a zero start time.
		{d: time.Duration(1<<63 - 1), want: "106751d 23h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, humanizeDuration(tt.d))
		})
	}
}
