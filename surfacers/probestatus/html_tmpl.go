// Copyright 2022 The Cloudprober Authors.
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

package probestatus

import "github.com/cloudprober/cloudprober/web/resources"

var probeStatusTmpl = `
<html>
<!DOCTYPE html>
<meta charset="utf-8">

<head>
` + resources.Style + `
</head>

<body>
<b>Started</b>: {{.StartTime}} -- up {{.Uptime}}<br/>
<b>Version</b>: {{.Version}}<br>
<b>Config</b>: <a href="/config">/config</a><br>

{{$durations := .Durations}}
{{$probesStatus := .ProbesStatus}}
{{$probesStatusDebug := .ProbesStatusDebug}}

<h3> Success Ratio </h3>
{{range $probeName := .ProbeNames}}
<p>
  <b>Probe: {{$probeName}}</b><br>

  <table class="status-list">
    <tr><td></td>
    {{range $durations}}
      <td><b>{{.}}</b></td>
    {{end}}
    </tr>

    {{index $probesStatus .}}
  </table>
</p>
<div id="chart_{{$probeName}}"></div>
{{end}}

<hr>
<div class="debugging">
  <b>Debugging Info</b>
  <br>
  {{range $probeName := .ProbeNames}}
    <p>
      <b>Probe: {{$probeName}}</b><br>

      {{index $probesStatusDebug $probeName}}
    </p>
  {{end}}
</div>

</html>
`
