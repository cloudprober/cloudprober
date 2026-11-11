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

var htmlTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">

<title>{{.Title}}</title>

<script>
  var startTime = '{{.StartTime}}';
  var maxDurationSec = {{.MaxDurationSec}};
  var allProbes = [];
  {{range $probeName := .AllProbes}}
    allProbes.push('{{$probeName}}');
  {{end}}
</script>

<link href="{{.BaseURL}}/static/c3.min.css" rel="stylesheet">
{{.HeadLinks}}
<script src="{{.BaseURL}}/static/jquery-3.6.0.min.js" charset="utf-8"></script>
<script src="{{.BaseURL}}/static/d3.v5.min.js" charset="utf-8"></script>
<script src="{{.BaseURL}}/static/c3.min.js" charset="utf-8"></script>
<script src="{{.BaseURL}}/static/probestatus.js" charset="utf-8"></script>

<script>
var d = {};
var psd = {};

{{$graphData := .GraphData}}

{{range $probeName := .ProbeNames}}
psd['{{$probeName}}'] = {{index $graphData .}}
{{end}}

populateD();
</script>
</head>

<body>
  {{.Header}}

  <div class="time-picker">
    <button type="button" id="tp-back" class="tp-btn" title="Move back">&#9664;</button>
    <button type="button" id="tp-toggle" class="tp-btn" aria-haspopup="true" aria-expanded="false"></button>
    <button type="button" id="tp-fwd" class="tp-btn" title="Move forward">&#9654;</button>
    <div id="tp-panel" class="tp-panel" hidden>
      <form id="tp-custom" class="tp-custom">
        <div class="tp-heading">Custom range</div>
        <label>Duration<input id="tp-duration" type="text" placeholder="e.g. 90m, 6h"></label>
        <label>Ending<input id="tp-endtime" type="datetime-local"></label>
        <div class="tp-hint">Leave a field empty for its default: all data, ending now.</div>
        <div id="tp-error" class="tp-error"></div>
        <button type="submit">Apply</button>
      </form>
      <div class="tp-quick">
        <div class="tp-heading">Quick ranges</div>
        <ul id="tp-quick-list"></ul>
      </div>
    </div>
  </div>
  <div style="clear: both; padding-top: 10px"></div>

{{$durations := .Durations}}
{{$statusTable := .StatusTable}}
{{$debugData := .DebugData}}

{{with .Summary}}
<div class="summary-card">
<h3>Summary (last {{$.SummaryWindow}})</h3>
<table class="probe-summary">
  {{range .}}
  <tr>
    <td><span class="health-dot health-{{.Health}}" title="{{.Health}}"></span></td>
    <td><a href="#probe-{{.Name}}">{{.Name}}</a></td>
    <td>{{if .Total}}{{.OK}}/{{.Total}} targets OK{{else}}<span class="greyed">No recent data</span>{{end}}</td>
    <td>{{range $i, $t := .Failing}}{{if $i}}, {{end}}{{$t}}{{end}}{{if .MoreFailing}} +{{.MoreFailing}} more{{end}}</td>
    {{if $.LogsURL}}<td><a class="logs-link" href="{{$.LogsURL}}?source={{.Name}}">logs</a></td>{{end}}
  </tr>
  {{end}}
</table>
</div>
{{end}}

<h3> Success Ratio </h3>
<div id="probe-selector" style="background: #E1F6FF; padding: 5px 5px; border-radius: 2px 2px"></div>

{{range $probeName := .ProbeNames}}
<p>
  <b id="probe-{{$probeName}}">Probe: {{$probeName}}</b>
  {{if $.LogsURL}}<a class="logs-link" href="{{$.LogsURL}}?source={{$probeName}}">logs</a>{{end}}<br>

  <table class="status-list">
    <tr><td></td>
    {{range $durations}}
      <td><b>{{.}}</b></td>
    {{end}}
    </tr>

    {{index $statusTable .}}
  </table>
</p>
<div id="chart_{{$probeName}}"></div>
{{end}}

<div id="debug-info" style="display:none">
  <hr>
  <b>Debugging Info:</b>
  <br>
  {{range $probeName := .ProbeNames}}
    <p>
      <b>Probe: {{$probeName}}</b><br>

      {{index $debugData $probeName}}
    </p>
  {{end}}
</div>

<script>
for (const probe in d) {
  c3.generate(d[probe]);
}
</script>
</html>
`
