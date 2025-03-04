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

var htmlTmpl = `
<html>
<!DOCTYPE html>
<meta charset="utf-8">

<head>

<script>
  var startTime = '{{.StartTime}}';
  var allProbes = [];
  {{range $probeName := .AllProbes}}
    allProbes.push('{{$probeName}}');
  {{end}}
</script>

<link href="{{.BaseURL}}/static/c3.min.css" rel="stylesheet">
<link href="{{.LinkPrefix}}/static/cloudprober.css" rel="stylesheet">
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

  <div style="float:right" class="graph-options">
    <label style="font-weight:bold" for="graph-duration">Graph Duration:</label>
    <input type="text" list="graph-duration-list" id="graph-duration" style="width: 50px;"/>
    <datalist name="graph-duration-list" id="graph-duration-list" >
      <option>15m</option>
      <option>3h</option>
      <option>6h</option>
      <option>12h</option>
      <option>24h</option>
      <option>72h</option>
    </datalist>
    <label style="font-weight:bold" for="graph-endtime">Endtime:</label>
    <input type="datetime-local" id="graph-endtime" name="graph-endtime" style="width: 200px;">
  </div>
  <br><br><br><br>

{{$durations := .Durations}}
{{$statusTable := .StatusTable}}
{{$debugData := .DebugData}}

<h3> Success Ratio </h3>
<div id="probe-selector" style="background: #E1F6FF; padding: 5px 5px; border-radius: 2px 2px"></div>

{{range $probeName := .ProbeNames}}
<p>
  <b>Probe: {{$probeName}}</b><br>

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
  var chart = c3.generate(d[probe]);

  setTimeout(function () {
      chart.load();
  }, 1000);
}
</script>
</html>
`
