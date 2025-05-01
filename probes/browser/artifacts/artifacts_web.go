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

package artifacts

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/web/resources"
)

var dateDirFormat = regexp.MustCompile("[0-9]{4}-[0-9]{2}-[0-9]{2}")

type dateDir struct {
	DateDir   string
	Timestamp []string
}

func rootLinkPrefix(currentPath string) string {
	numSegments := len(strings.Split(strings.Trim(currentPath, "/"), "/"))
	linkPrefix := ""
	for range numSegments {
		linkPrefix += "../"
	}
	return linkPrefix
}

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
  </style>
</head>
<body>
%s
<div style="display: block; clear: both; padding-top: 10px">
<hr>
<ul>
{{ range . }}
 {{ $dateDir := .DateDir }}
 <li><a href="tree/{{ $dateDir }}">{{ $dateDir }}</a></li>
<ul>
{{ range .Timestamp }}
<li><a href="tree/{{ $dateDir }}/{{.}}">{{.}}</a></li>
{{ end }}
</ul>
{{ end }}
</ul></div></body></html>`, linkPrefix, resources.Header(linkPrefix))))
}

func stripTreePrefix(basePath string, global bool, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// strip upto basePath
		// e.g. /artifacts/probe1/tree/test.txt -> /probe1/tree/test.txt for
		// global where basePath is typically /artifacts
		// e.g. /artifacts/probe1/tree/test.txt -> /tree/test.txt for probe
		// level where basePath is /artifacts/probe1
		relURLPath := strings.TrimPrefix(r.URL.Path, basePath)

		urlParts := strings.Split(strings.TrimPrefix(relURLPath, "/"), "/")
		from, to := "", ""
		if global {
			// For global, after removing basePath, path will look like
			// probe1/tree/test.txt
			if len(urlParts) < 2 || urlParts[1] != "tree" {
				http.NotFound(w, r)
				return
			}
			from, to = path.Join(basePath, urlParts[0], "tree"), "/"+urlParts[0]
		} else {
			// For probe level, after removing basePath, path will look like
			// /tree/test.txt
			if len(urlParts) < 1 || urlParts[0] != "tree" {
				http.NotFound(w, r)
				return
			}
			from, to = path.Join(basePath, "tree"), ""
		}

		// Following is based on Go's http.StripPrefix works. We're not using
		// http.StripPrefix because we've to handle cases when 'tree' is not
		// the 1st segment of the path -- based on the config, 1st segment can
		// be the probe name.
		p := strings.Replace(r.URL.Path, from, to, 1)
		rp := strings.Replace(r.URL.RawPath, from, to, 1)
		if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			h.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
		}
	})
}

func smartViewHandler(w http.ResponseWriter, r *http.Request, rootDir string) {
	tsDirs, err := getTimestampDirectories(rootDir, time.Time{}, time.Time{}, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	dirsMap := make(map[string]*dateDir)
	var dirsList []*dateDir
	for _, dir := range tsDirs {
		ddKey := filepath.Base(filepath.Dir(dir.Path))
		if dirsMap[ddKey] == nil {
			dirsMap[ddKey] = &dateDir{
				DateDir:   ddKey,
				Timestamp: []string{},
			}
			dirsList = append(dirsList, dirsMap[ddKey])
		}
		dirsMap[ddKey].Timestamp = append(dirsMap[ddKey].Timestamp, filepath.Base(dir.Path))
	}
	if err := tsDirTmpl(r.URL.Path).ExecuteTemplate(w, "tsDirTmpl", dirsList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	return
}

func serveArtifacts(path, root string, global bool) error {
	path = strings.TrimRight(path, "/")

	if path == "" {
		return fmt.Errorf("artifacts web server path cannot be empty")
	}

	// Set up tree view handler
	probeTreePath := path + "/tree/"
	if global {
		probeTreePath = path + "/{probeName}/tree/"
	}
	if err := state.AddWebHandler(probeTreePath, stripTreePrefix(path, global, http.FileServer(http.Dir(root))).ServeHTTP); err != nil {
		return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
	}

	patternPath := path + "/{$}"
	if global {
		patternPath = path + "/{probeName}/{$}"
	}
	if err := state.AddWebHandler(patternPath, func(w http.ResponseWriter, r *http.Request) {
		dirBase := root
		if global {
			dirBase = filepath.Join(root, r.PathValue("probeName"))
		}
		smartViewHandler(w, r, dirBase)
	}); err != nil {
		return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
	}

	// For global, add a handler for the root path as well
	if global {
		if err := state.AddWebHandler(path+"/{$}", http.StripPrefix(path, http.FileServer(http.Dir(root))).ServeHTTP); err != nil {
			return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
		}
	}
	return nil
}
