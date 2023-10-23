// Copyright 2023 The Cloudprober Authors.
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

// Package webutils provides web related utilities for cloudprober.
package webutils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHandled(t *testing.T) {
	srvMux := http.NewServeMux()

	srvMux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {})
	srvMux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {})
	srvMux.Handle("/", http.RedirectHandler("/status", http.StatusFound))

	tests := map[string]bool{
		"/":            true,
		"/probestatus": false,
		"/status":      true,
		"/config":      true,
		"/config2":     false,
	}

	for url, wantResult := range tests {
		assert.Equal(t, wantResult, IsHandled(srvMux, url))
	}
}
