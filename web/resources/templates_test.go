// Copyright 2026 The Cloudprober Authors.
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
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/state"
	"github.com/stretchr/testify/assert"
)

func TestExecTmpl(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    *template.Template
		data    interface{}
		wantStr string
		wantErr bool
	}{
		{
			name:    "simple template",
			tmpl:    template.Must(template.New("test").Parse("hello {{.}}")),
			data:    "world",
			wantStr: "hello world",
		},
		{
			name:    "template execution error",
			tmpl:    template.Must(template.New("test").Parse("{{.MissingMethod}}")),
			data:    "string-has-no-MissingMethod",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExecTmpl(tt.tmpl, tt.data)
			if tt.wantErr {
				// On error, the output should be HTML-escaped error text
				assert.NotEmpty(t, got, "error output should not be empty")
				// Should not contain raw HTML (error is escaped)
				assert.NotContains(t, string(got), "<template>")
			} else {
				assert.Equal(t, template.HTML(tt.wantStr), got)
			}
		})
	}
}

func TestLinkPrefix(t *testing.T) {
	urlToExpLinkPrefix := map[string]string{
		"/":                 "",
		"":                  "",
		"/artifacts/probe1": "../../",
		"/probe1":           "../",
	}
	for url, expLinkPrefix := range urlToExpLinkPrefix {
		if got := RootLinkPrefix(url); got != expLinkPrefix {
			t.Errorf("LinkPrefix(%q) = %q, want %q", url, got, expLinkPrefix)
		}
	}
}

func TestRenderPage(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want string
	}{
		{
			name: "root path",
			path: "/",
			body: "test body",
			want: `
<html>
<head>
  <link href="static/cloudprober.css" rel="stylesheet">
</head>

<body>

<header>
  <a href="https://cloudprober.org">Cloudprober</a> (<a href="https://github.com/cloudprober/cloudprober">Github</a>)
</header> 
<hr/>
<div style="float:left">
  <b>Started</b>: 0001-01-01 00:00:00 &#43;0000 UTC -- up 2562047h47m16.854s<br/>
  <b>Version</b>: v1.0.0<br>
  <b>Built at</b>: 0001-01-01 00:00:00 &#43;0000 UTC<br>
  <b>Other Links </b>(<a href="links">all</a>):
  	<a href="status">/status</a>,
	<a href="config-running">/config</a> (<a href="config-parsed">parsed</a> | <a href="config">raw</a>),
	
	
	<a href="alerts">/alerts</a>
</div>

<br><br><br><br>
test body
</body>
</html>
`,
		},
		{
			name: "deep path",
			path: "/a/b",
			body: "test body deep",
			want: `
<html>
<head>
  <link href="../../static/cloudprober.css" rel="stylesheet">
</head>

<body>

<header>
  <a href="https://cloudprober.org">Cloudprober</a> (<a href="https://github.com/cloudprober/cloudprober">Github</a>)
</header> 
<hr/>
<div style="float:left">
  <b>Started</b>: 0001-01-01 00:00:00 &#43;0000 UTC -- up 2562047h47m16.854s<br/>
  <b>Version</b>: v1.0.0<br>
  <b>Built at</b>: 0001-01-01 00:00:00 &#43;0000 UTC<br>
  <b>Other Links </b>(<a href="../../links">all</a>):
  	<a href="../../status">/status</a>,
	<a href="../../config-running">/config</a> (<a href="../../config-parsed">parsed</a> | <a href="../../config">raw</a>),
	
	
	<a href="../../alerts">/alerts</a>
</div>

<br><br><br><br>
test body deep
</body>
</html>
`,
		},
	}

	oldVersion := state.Version()
	defer state.SetVersion(oldVersion)
	state.SetVersion("v1.0.0")
	state.SetBuildTimestamp(time.Time{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderPage(tt.path, tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}
