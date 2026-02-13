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
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudprober/cloudprober/state"
)

func rootLinkPrefix(currentPath string) string {
	if currentPath == "" || currentPath == "/" {
		return ""
	}
	numSegments := len(strings.Split(strings.Trim(currentPath, "/"), "/"))
	linkPrefix := ""
	for range numSegments {
		linkPrefix += "../"
	}
	return linkPrefix
}

// substitutionForTreePath computes URL substitutions (from, to) for tree view.
// e.g. for urlPath "/artifacts/probe1/tree/test.txt",
// for global,      from:"/artifacts/probe1/tree", to: "/probe1"
// for probe-level, from:"/artifacts/probe1/tree", to: "/"
func substitutionForTreePath(urlPath, basePath string, global bool) (string, string, error) {
	relURLPath := strings.TrimPrefix(urlPath, basePath)

	urlParts := strings.Split(strings.TrimPrefix(relURLPath, "/"), "/")
	if global {
		// After removing basePath, path will look like probe1/tree/test.txt
		if len(urlParts) < 2 || urlParts[1] != "tree" {
			return "", "", fmt.Errorf("invalid path: %s, expected format: /{basePath}/{probeName}/tree/{filepath}", urlPath)
		}
		return path.Join(basePath, urlParts[0], "tree"), "/" + urlParts[0], nil
	}

	// For probe level, path will look like /tree/test.txt
	if len(urlParts) < 1 || urlParts[0] != "tree" {
		return "", "", fmt.Errorf("invalid path: %s, expected format: /{basePath}/tree/{filepath}", urlPath)
	}
	return path.Join(basePath, "tree"), "/", nil
}

// stripTreePrefix is based on how Go's http.StripPrefix works. We're not using
// http.StripPrefix because we've to handle cases when 'tree' is not the 1st
// segment of the path -- based on the config, 1st segment can be the probe name.
func stripTreePrefix(basePath string, global bool, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		from, to, err := substitutionForTreePath(r.URL.Path, basePath, global)
		if err != nil {
			http.NotFound(w, r)
			return
		}
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

type tmplTSDir struct {
	Timestamp string
	TimeStr   string
	Failed    bool
}

type tmplDateData struct {
	DateDir string
	TSDirs  []tmplTSDir
}

func tmplData(tsDirs []DirEntry) []*tmplDateData {
	dirsMap := make(map[string]*tmplDateData)
	var dirsList []*tmplDateData
	for _, dir := range tsDirs {
		ddKey := filepath.Base(filepath.Dir(dir.Path))
		if dirsMap[ddKey] == nil {
			dirsMap[ddKey] = &tmplDateData{
				DateDir: ddKey,
				TSDirs:  []tmplTSDir{},
			}
			dirsList = append(dirsList, dirsMap[ddKey])
		}
		dirsMap[ddKey].TSDirs = append(dirsMap[ddKey].TSDirs, tmplTSDir{
			Timestamp: filepath.Base(dir.Path),
			TimeStr:   dir.ModTime.Format("15:04:05 MST"),
			Failed:    dir.Failed,
		})
	}
	return dirsList
}

func smartViewHandler(w http.ResponseWriter, r *http.Request, rootDir string) {
	tsDirs, err := getTimestampDirectories(rootDir, r.URL.Query(), 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if err := tsDirTmpl(r.URL.Path).ExecuteTemplate(w, "tsDirTmpl", tmplData(tsDirs)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	return
}

func ServeArtifacts(path, root string, global bool) error {
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

	var artifactsLinkOpts []state.HandlerOption
	if !global {
		artifactsLinkOpts = append(artifactsLinkOpts, state.WithArtifactsLink(path))
	}

	if err := state.AddWebHandler(patternPath, func(w http.ResponseWriter, r *http.Request) {
		dirBase := root
		if global {
			dirBase = filepath.Join(root, r.PathValue("probeName"))
		}
		smartViewHandler(w, r, dirBase)
	}, artifactsLinkOpts...); err != nil {
		return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
	}

	// For global, add a handler for the root path as well
	if global {
		if err := state.AddWebHandler(path+"/{$}", http.StripPrefix(path, http.FileServer(http.Dir(root))).ServeHTTP, state.WithArtifactsLink(path+"/")); err != nil {
			return fmt.Errorf("error adding web handler for artifacts web server: %v", err)
		}
	}

	return nil
}
