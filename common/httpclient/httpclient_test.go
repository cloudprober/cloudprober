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

package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckRedirectFunc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/1", http.StatusTemporaryRedirect)
		case "/1":
			http.Redirect(w, r, "/2", http.StatusTemporaryRedirect)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	cases := []struct {
		desc       string
		n          int
		wantStatus int
	}{
		{"zero blocks all redirects", 0, http.StatusTemporaryRedirect},
		{"one allows first redirect only", 1, http.StatusTemporaryRedirect},
		{"two follows both redirects", 2, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			c := &http.Client{CheckRedirect: CheckRedirectFunc(tc.n)}
			resp, err := c.Get(ts.URL)
			assert.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
