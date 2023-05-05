// Copyright 2022-2023 The Cloudprober Authors.
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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

const (
	largeBodyThreshold = bytes.MinRead // 512.
)

func IsHandled(mux *http.ServeMux, url string) bool {
	_, matchedPattern := mux.Handler(httptest.NewRequest("", url, nil))
	return matchedPattern == url
}

// RequestBody returns a reusable HTTP request body if its size is smaller than
// largeBodyThreshold. This is the most common use case in probing. If the body
// is larger than largeBodyThreshold, it returns a buffered reader. We do that
// because HTTP transport reads limited bytes at a time.
type RequestBody struct {
	b  []byte
	ct string
}

// NewRequestBody returns a new RequestBody object.
func NewRequestBody(data ...string) *RequestBody {
	if len(data) == 0 {
		return nil
	}

	dataBytes := []byte(strings.Join(data, "&"))
	rb := &RequestBody{
		b: dataBytes,
	}

	if len(dataBytes) <= largeBodyThreshold {
		rb.ct = contentType(data)
	}

	return rb
}

func (rb *RequestBody) Read(p []byte) (int, error) {
	return copy(p, rb.b), io.EOF
}

func (rb *RequestBody) Reader() io.ReadCloser {
	if rb == nil {
		return nil
	}
	if len(rb.b) > largeBodyThreshold {
		return io.NopCloser(bytes.NewReader(rb.b))
	}
	return io.NopCloser(rb)
}

func (rb *RequestBody) Buffered() bool {
	return rb != nil && len(rb.b) > largeBodyThreshold
}

func (rb *RequestBody) Len() int64 {
	if rb == nil {
		return 0
	}
	return int64(len(rb.b))
}

func (rb *RequestBody) ContentType() string {
	if rb == nil {
		return ""
	}
	return rb.ct
}

func contentType(data []string) string {
	// If there is only one data element, return json content type if data is
	// valid JSON.
	if len(data) == 1 {
		if json.Valid([]byte(data[0])) {
			return "application/json"
		}
	}

	if _, err := url.ParseQuery(strings.Join(data, "&")); err == nil {
		return "application/x-www-form-urlencoded"
	}
	return ""
}

func HTTPRequest(method, url string, data []string, headers map[string]string) (*http.Request, error) {
	body := NewRequestBody(data...)

	req, err := http.NewRequest(method, url, body.Reader())
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}

	if body.ContentType() != "" {
		req.Header.Set("Content-Type", body.ContentType())
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}
