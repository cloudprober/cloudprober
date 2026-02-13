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
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

func RenderPage(path, body string) string {
	linkPrefix := RootLinkPrefix(path)
	header := Header(linkPrefix)
	return fmt.Sprintf(`
<html>
<head>
  <link href="%sstatic/cloudprober.css" rel="stylesheet">
</head>

<body>
%s
<br><br><br><br>
%s
</body>
</html>
`, linkPrefix, header, body)
}

type linksData struct {
	Title string
	Links []string
}

var linksTmpl = template.Must(template.New("allLinks").Parse(`
<h3>{{.Title}}:</h3>
<ul>
  {{ range .Links}}
  {{ $link := . }}
  {{ if eq $link "" }} {{ $link = "/" }} {{ end }}
  <li><a href="{{ $link }}">{{ $link }}</a></li>
  {{ end }}
</ul>
`))

func ExecTmpl(tmpl *template.Template, v interface{}) template.HTML {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, v)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(err.Error()))
	}
	return template.HTML(buf.String())
}

func RootLinkPrefix(currentPath string) string {
	if currentPath == "" || currentPath == "/" {
		return ""
	}
	numSegments := len(strings.Split(strings.Trim(currentPath, "/"), "/"))
	out := ""
	for range numSegments {
		out += "../"
	}
	return out
}

func LinksPage(path, title string, links []string) string {
	return RenderPage(path, string(ExecTmpl(linksTmpl, linksData{Title: title, Links: links})))
}
