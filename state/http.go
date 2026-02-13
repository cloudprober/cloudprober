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

package state

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
)

// SetDefaultHTTPServeMux stores the default HTTP ServeMux in state. This
// allows other modules to add their own handlers to the common ServeMux.
func SetDefaultHTTPServeMux(mux *http.ServeMux) {
	st.Lock()
	defer st.Unlock()
	st.httpServeMux = mux
	st.webURLs = make([]string, 0)
	st.artifactsURLs = make([]string, 0)
}

// DefaultHTTPServeMux returns the default HTTP ServeMux.
func DefaultHTTPServeMux() *http.ServeMux {
	st.RLock()
	defer st.RUnlock()
	return st.httpServeMux
}

func IsHandled(url string) bool {
	st.Lock()
	defer st.Unlock()

	if st.httpServeMux == nil {
		return false
	}

	_, matchedPattern := st.httpServeMux.Handler(httptest.NewRequest("", url, nil))
	return matchedPattern == url
}

type handlerOptions struct {
	isArtifact       bool
	artifactLinkPath string
}

type HandlerOption func(*handlerOptions)

func WithArtifactsLink(linkPath string) HandlerOption {
	return func(o *handlerOptions) {
		o.isArtifact = true
		o.artifactLinkPath = linkPath
	}
}

func AddWebHandler(path string, f func(w http.ResponseWriter, r *http.Request), opts ...HandlerOption) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	hOptions := &handlerOptions{}
	for _, opt := range opts {
		opt(hOptions)
	}

	st.Lock()
	defer st.Unlock()

	if st.httpServeMux == nil {
		return errors.New("default http server not initialized")
	}

	if slices.Contains(st.webURLs, path) {
		return fmt.Errorf("path %s already registered", path)
	}

	st.webURLs = append(st.webURLs, path)
	if hOptions.isArtifact {
		link := path
		if hOptions.artifactLinkPath != "" {
			link = hOptions.artifactLinkPath
		}
		st.artifactsURLs = append(st.artifactsURLs, link)
	}
	st.httpServeMux.HandleFunc(path, f)

	return nil
}

func AllLinks() []string {
	st.RLock()
	defer st.RUnlock()

	return st.webURLs
}
