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

// Package resources provides webpages related resources.
package resources

import (
	"bytes"
	"html/template"
	"time"

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/sysvars"
)

var t = template.Must(template.New("header").Parse(`
<header>
  <a href="https://cloudprober.org">Cloudprober</a> (<a href="https://github.com/cloudprober/cloudprober">Github</a>)
</header> 
<hr/>
<div style="float:left">
  <b>Started</b>: {{.StartTime}} -- up {{.Uptime}}<br/>
  <b>Version</b>: {{.Version}}<br>
  <b>Built at</b>: {{.BuiltAt}}<br>
  <b>Other Links</b>: <a href="/status">/status</a>, <a href="/config-running">/config</a> (<a href="/config-parsed">parsed</a> | <a href="/config">raw</a>), <a href="/alerts">/alerts</a>, <a href="/health">/health</a><br>
</div>
`))

func Header() template.HTML {
	var buf bytes.Buffer

	startTime := sysvars.StartTime().Truncate(time.Millisecond)
	uptime := time.Since(startTime).Truncate(time.Millisecond)

	t.Execute(&buf, struct {
		Version, BuiltAt, StartTime, Uptime, RightDiv interface{}
	}{
		Version:   runconfig.Version(),
		BuiltAt:   runconfig.BuildTimestamp(),
		StartTime: startTime,
		Uptime:    uptime,
	})

	return template.HTML(buf.String())
}
