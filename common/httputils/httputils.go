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

// Package httputils implements HTTP related utilities.
package httputils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

func IsHandled(mux *http.ServeMux, url string) bool {
	_, matchedPattern := mux.Handler(httptest.NewRequest("", url, nil))
	return matchedPattern == url
}

// RequestBody encapsulates the request body and implements the io.Reader()
// interface.
type RequestBody struct {
	b []byte
}

// NewRequestBody returns a new RequestBody object.
func NewRequestBody(data []byte) *RequestBody {
	return &RequestBody{b: data}
}

// Read implements the io.Reader interface. Instead of using buffered read,
// it simply copies the bytes to the provided slice in one go (depending on
// the input slice capacity) and returns io.EOF. Buffered reads require
// resetting the buffer before re-use, restricting our ability to reuse the
// request object and using it concurrently.
func (rb *RequestBody) Read(p []byte) (int, error) {
	return copy(p, rb.b), io.EOF
}

func setContentType(req *http.Request, data []string) {
	if len(data) == 1 {
		if json.Valid([]byte(data[0])) {
			req.Header.Set("Content-Type", "application/json")
			return
		}
	}
	if _, err := url.ParseQuery(strings.Join(data, "&")); err == nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return
	}
}

func HTTPRequest(method, url string, data []string, headers map[string]string) (*http.Request, error) {
	body := &RequestBody{b: []byte(strings.Join(data, "&"))}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	setContentType(req, data)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}
