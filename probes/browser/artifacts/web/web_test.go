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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/state"
	"github.com/stretchr/testify/assert"
)

func TestTmplData(t *testing.T) {
	// 2025-05-19T18:28:15-07:00 is the current time reference
	tsDirs := []DirEntry{
		{Path: "/tmp/2025-05-18/1234", ModTime: time.Date(2025, 5, 18, 10, 11, 12, 0, time.FixedZone("PDT", -7*3600)), Failed: false},
		{Path: "/tmp/2025-05-18/5678", ModTime: time.Date(2025, 5, 18, 11, 22, 33, 0, time.FixedZone("PDT", -7*3600)), Failed: true},
		{Path: "/tmp/2025-05-19/9999", ModTime: time.Date(2025, 5, 19, 9, 0, 0, 0, time.FixedZone("PDT", -7*3600)), Failed: false},
	}

	result := tmplData(tsDirs)
	assert.Equal(t, 2, len(result), "should group by date directories")

	// Find 2025-05-18 group
	var group18, group19 *tmplDateData
	for _, g := range result {
		if g.DateDir == "2025-05-18" {
			group18 = g
		} else if g.DateDir == "2025-05-19" {
			group19 = g
		}
	}
	if assert.NotNil(t, group18, "2025-05-18 group exists") {
		assert.Equal(t, 2, len(group18.TSDirs), "should have 2 entries for 2025-05-18")
		assert.Equal(t, "1234", group18.TSDirs[0].Timestamp)
		assert.Equal(t, "10:11:12 PDT", group18.TSDirs[0].TimeStr)
		assert.False(t, group18.TSDirs[0].Failed)
		assert.Equal(t, "5678", group18.TSDirs[1].Timestamp)
		assert.Equal(t, "11:22:33 PDT", group18.TSDirs[1].TimeStr)
		assert.True(t, group18.TSDirs[1].Failed)
	}
	if assert.NotNil(t, group19, "2025-05-19 group exists") {
		assert.Equal(t, 1, len(group19.TSDirs), "should have 1 entry for 2025-05-19")
		assert.Equal(t, "9999", group19.TSDirs[0].Timestamp)
		assert.Equal(t, "09:00:00 PDT", group19.TSDirs[0].TimeStr)
		assert.False(t, group19.TSDirs[0].Failed)
	}
}

func TestSubstitutionForTreePath(t *testing.T) {
	tests := []struct {
		name     string
		urlPath  string
		basePath string
		global   bool
		expFrom  string
		expTo    string
		expErr   bool
	}{
		{
			name:     "global valid path",
			urlPath:  "/artifacts/probe1/tree/test.txt",
			basePath: "/artifacts/",
			global:   true,
			expFrom:  "/artifacts/probe1/tree",
			expTo:    "/probe1",
			expErr:   false,
		},
		{
			name:     "probe valid path",
			urlPath:  "/artifacts/probe1/tree/test.txt",
			basePath: "/artifacts/probe1",
			global:   false,
			expFrom:  "/artifacts/probe1/tree",
			expTo:    "/",
			expErr:   false,
		},
		{
			name:     "global invalid path (missing tree)",
			urlPath:  "/artifacts/probe1/notree/test.txt",
			basePath: "/artifacts/",
			global:   true,
			expErr:   true,
		},
		{
			name:     "global invalid path (wrong place)",
			urlPath:  "/artifacts/tree/test.txt",
			basePath: "/artifacts/",
			global:   true,
			expErr:   true,
		},
		{
			name:     "probe invalid path (missing tree)",
			urlPath:  "/artifacts/notree/test.txt",
			basePath: "/artifacts/",
			global:   false,
			expErr:   true,
		},
		{
			name:     "probe invalid path (wrong place)",
			urlPath:  "/probe1/x/tree/test.txt",
			basePath: "/probe1",
			global:   false,
			expErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, prefix, err := substitutionForTreePath(tt.urlPath, tt.basePath, tt.global)
			if tt.expErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if path != tt.expFrom {
				t.Errorf("expected path %q, got %q", tt.expFrom, path)
			}
			if prefix != tt.expTo {
				t.Errorf("expected prefix %q, got %q", tt.expTo, prefix)
			}
		})
	}
}

func verifyWebServerResponse(t *testing.T, mux *http.ServeMux, path string, expectedStatus int, expectedBody string) {
	req, err := http.NewRequest("GET", path, nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, expectedStatus, rr.Code)
	if expectedBody != "-" {
		assert.Equal(t, expectedBody, rr.Body.String(), "path: %s", path)
	}
}

func TestServeArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, "probe1"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "probe1", "test.txt"), []byte("test"), 0644))
	assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, "probe2"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "probe2", "test.txt"), []byte("test"), 0644))

	oldSrvMux := state.DefaultHTTPServeMux()
	defer func() {
		state.SetDefaultHTTPServeMux(oldSrvMux)
	}()

	mux := http.NewServeMux()
	state.SetDefaultHTTPServeMux(mux)

	// Serve a simple status endpoint
	state.AddWebHandler("/status/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("status"))
	})

	tests := []struct {
		name              string
		path              string
		globalPath        string
		root              string
		globalRoot        string
		url               []string
		expectedPattern   []string
		expectedBody      []string
		expectedArtifacts []string
		expectError       bool
	}{
		{
			name:        "empty path",
			path:        "",
			root:        tmpDir,
			expectError: true,
		},
		{
			name:        "existing path",
			path:        "/status/",
			root:        tmpDir,
			expectError: true,
		},
		{
			name:       "valid path and root",
			path:       "/artifacts/probe1",
			globalPath: "/artifacts",
			root:       filepath.Join(tmpDir, "probe1"),
			globalRoot: tmpDir,
			url: []string{
				"/artifacts/probe1/tree/test.txt",
				"/artifacts/probe1/",
				"/artifacts/probe2/tree/test.txt", // handled by global
				"/artifacts/probe2/",              // handled by global
				"/artifacts/",                     // global root
			},
			expectedPattern: []string{
				"/artifacts/probe1/tree/",
				"/artifacts/probe1/{$}",
				"/artifacts/{probeName}/tree/", // handled by global
				"/artifacts/{probeName}/{$}",   // handled by global
				"/artifacts/{$}",               // global root
			},
			expectedBody:      []string{"test", "-", "test", "-", "-"},
			expectedArtifacts: []string{"/artifacts/", "/artifacts/probe1"},
			expectError:       false,
		},
		{
			name:        "same path, existing handler",
			path:        "/artifacts/probe1/",
			root:        tmpDir,
			expectError: true,
		},
		{
			name:              "path with trailing slash",
			path:              "/artifacts2/probe1/",
			root:              filepath.Join(tmpDir, "probe1"),
			url:               []string{"/artifacts2/probe1/tree/test.txt", "/artifacts2/probe1/"},
			expectedPattern:   []string{"/artifacts2/probe1/tree/", "/artifacts2/probe1/{$}"},
			expectedBody:      []string{"test", "-"},
			expectedArtifacts: []string{"/artifacts2/probe1"},
			expectError:       false,
		},
		{
			name:       "valid path and root - deeper",
			path:       "/x/y/probe1",
			globalPath: "/z/a",
			root:       filepath.Join(tmpDir, "probe1"),
			globalRoot: tmpDir,
			url: []string{
				"/x/y/probe1/tree/test.txt",
				"/x/y/probe1/",
				"/z/a/probe2/tree/test.txt", // handled by global
				"/z/a/probe2/",              // handled by global
			},
			expectedPattern: []string{
				"/x/y/probe1/tree/",
				"/x/y/probe1/{$}",
				"/z/a/{probeName}/tree/", // handled by global
				"/z/a/{probeName}/{$}",   // handled by global
			},
			expectedBody:      []string{"test", "-", "test", "-", "-"},
			expectedArtifacts: []string{"/x/y/probe1", "/z/a/"},

			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServeArtifacts(tt.path, tt.root, false)
			if tt.expectError {
				assert.Error(t, err)
				t.Log(err)
				return
			}

			if tt.globalPath != "" {
				err = ServeArtifacts(tt.globalPath, tt.globalRoot, true)
			}
			if tt.expectError {
				assert.Error(t, err)
				t.Log(err)
				return
			}

			assert.NoError(t, err)

			// Check if the handler was added
			for i, u := range tt.url {
				handler, pattern := mux.Handler(&http.Request{URL: &url.URL{Path: u}})
				assert.NotNil(t, handler)
				assert.Equal(t, tt.expectedPattern[i], pattern)
			}

			// Check if the file is served correctly
			for i, u := range tt.url {
				verifyWebServerResponse(t, mux, u, http.StatusOK, tt.expectedBody[i])
			}

			// For global paths, verify the root handler returns styled probe list
			if tt.globalPath != "" {
				req, err := http.NewRequest("GET", tt.globalPath+"/", nil)
				assert.NoError(t, err)
				rr := httptest.NewRecorder()
				mux.ServeHTTP(rr, req)
				body := rr.Body.String()
				assert.Contains(t, body, "<h3>Probes:</h3>", "global root should have styled probe list")
				assert.Contains(t, body, "cloudprober.css", "global root should include cloudprober CSS")
				assert.Contains(t, body, `<a href="probe1">probe1</a>`, "global root should list probe1")
				assert.Contains(t, body, `<a href="probe2">probe2</a>`, "global root should list probe2")
			}
		})
	}
}
