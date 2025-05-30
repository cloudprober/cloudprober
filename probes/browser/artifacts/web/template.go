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

package web

import (
	_ "embed"
	"fmt"
	"text/template"

	"github.com/cloudprober/cloudprober/web/resources"
)

//go:embed artifacts.js
var artifactsJS string

func tsDirTmpl(currentPath string) *template.Template {
	linkPrefix := rootLinkPrefix(currentPath)
	return template.Must(template.New("tsDirTmpl").Parse(fmt.Sprintf(`
<html>
<head>
  <link href="%sstatic/cloudprober.css" rel="stylesheet">
  <style>
    ul {
	  padding-left: 20px;
	  line-height: 1.6;
	}
	.selectors {
      background: #fff;
      padding: 2px;
      max-width: 600px;
    }
    .datetime-group {
      display: flex;
      flex-wrap: wrap;
    }
    .datetime-field {
      flex: 1;
      min-width: 200px;
    }
    .selectors label {
      font-size: 14px;
      color: #333;
      margin-bottom: 5px;
      display: block;
    }
    .selectors input[type="datetime-local"] {
      padding: 4px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 14px;
      box-sizing: border-box;
    }
    .selectors input[type="datetime-local"]:focus {
      outline: none;
      border-color: #007bff;
      box-shadow: 0 0 5px rgba(0, 123, 255, 0.3);
    }
    .error {
      color: #d32f2f;
      font-size: 12px;
      display: none;
      margin-top: 5px;
    }
    .error.show {
      display: block;
    }
    .failed {
      font-size: 10px;
      color: #d32f2f;
    }
  </style>
</head>
<body>
%s
<div style="display: block; clear: both; padding-top: 10px">
  <hr>
  <div class="selectors">
    <div class="datetime-group">
      <div class="datetime-field">
        <label for="start-datetime">Start Date and Time (<span class="time-zone-abbr">Local</span>)</label>
        <input type="datetime-local" id="start-datetime" required>
      </div>
      <div class="datetime-field">
        <label for="end-datetime">End Date and Time (<span class="time-zone-abbr">Local</span>)</label>
        <input type="datetime-local" id="end-datetime" required>
        <span class="error" id="error-message">End date and time must be after start</span>
      </div>
      <div>
        <label for="failure-only">Failure Only</label>
        <input type="checkbox" id="failure-only">
      </div>
    </div>
  </div>
<ul>
{{ range . }}
 {{ $dateDir := .DateDir }}
 <li><a href="tree/{{ $dateDir }}">{{ $dateDir }}</a></li>
<ul>
{{ range .TSDirs }}
<li>
  <a href="tree/{{ $dateDir }}/{{.Timestamp}}">{{.Timestamp}} ({{.TimeStr}})</a>
  {{if .Failed}}<span class="failed">failed</span>{{end}}
</li>
{{ end }}
</ul>
{{ end }}
</ul></div>
<script>
%s
</script>
</body>
</html>`, linkPrefix, resources.Header(linkPrefix), artifactsJS)))
}
