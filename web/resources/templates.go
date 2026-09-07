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

// HeadLinks returns the stylesheet and favicon links shared by all
// cloudprober pages. linkPrefix is the relative path to the root, as returned
// by RootLinkPrefix or LinkPrefixFromCurrentPath.
func HeadLinks(linkPrefix string) template.HTML {
	return template.HTML(fmt.Sprintf(`  <link href="%sstatic/cloudprober.css" rel="stylesheet">
  <link rel="icon" href="%sstatic/favicon.ico" sizes="32x32">
  <link rel="icon" href="%sstatic/cloudprober-icon.svg" type="image/svg+xml">`,
		linkPrefix, linkPrefix, linkPrefix))
}

func RenderPage(path string, body template.HTML) string {
	linkPrefix := RootLinkPrefix(path)
	header := Header(linkPrefix)
	return fmt.Sprintf(`
<html>
<head>
  <title>Cloudprober</title>
%s
</head>

<body>
%s
<div style="clear: both; padding-top: 10px"></div>
%s
</body>
</html>
`, HeadLinks(linkPrefix), header, body)
}

type linksData struct {
	Title string
	Links []string
}

var linksTmpl = template.Must(template.New("allLinks").Parse(`
<h3>{{.Title}}:</h3>
<ul>
  {{ range .Links}}
  {{ $link := or . "/" }}
  <li><a href="{{ $link }}">{{ $link }}</a></li>
  {{ end }}
</ul>
`))

func ExecTmpl(tmpl *template.Template, v any) template.HTML {
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
	return RenderPage(path, ExecTmpl(linksTmpl, linksData{Title: title, Links: links}))
}
