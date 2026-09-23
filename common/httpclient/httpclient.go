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

// Package httpclient holds helpers shared by probes that build their own
// *http.Client.
package httpclient

import "net/http"

// CheckRedirectFunc returns a CheckRedirect that follows up to n redirects
// (n=0 disables). Uses ErrUseLastResponse so the redirect response is
// returned to the caller instead of surfacing as an error.
func CheckRedirectFunc(n int) func(*http.Request, []*http.Request) error {
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) > n {
			return http.ErrUseLastResponse
		}
		return nil
	}
}
