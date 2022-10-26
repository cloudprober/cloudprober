// Copyright 2018-2022 The Cloudprober Authors.
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

import "github.com/cloudprober/cloudprober/web/resources"

var parsedConfigTmpl = `
<html>

<head>
` + resources.Style + `
</head>

<b>Started</b>: {{.StartTime}} -- up {{.Uptime}}<br/>
<b>Version</b>: {{.Version}}<br>
<b>Other Links</b>: <a href="/config">/config</a> (<a href="/config-parsed">parsed</a>), <a href="/status">/status</a><br>

<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
`
