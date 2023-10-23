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
package httpreq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	configpb "github.com/cloudprober/cloudprober/internal/httpreq/proto"
)

const (
	largeBodyThreshold = bytes.MinRead // 512.
)

// RequestBody implements an HTTP request body.
type RequestBody struct {
	b  []byte
	ct string
}

// NewRequestBody returns a new RequestBody object.
func NewRequestBody(data ...string) *RequestBody {
	rb := &RequestBody{
		b: []byte(strings.Join(data, "&")),
	}

	if len(rb.b) <= largeBodyThreshold {
		rb.ct = contentType(data)
	}

	return rb
}

func (rb *RequestBody) Read(p []byte) (int, error) {
	return copy(p, rb.b), io.EOF
}

func (rb *RequestBody) Reader() io.ReadCloser {
	if rb == nil || len(rb.b) == 0 {
		return nil
	}
	return io.NopCloser(bytes.NewReader(rb.b))
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
	if len(data) == 0 {
		return ""
	}
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

// NewRequest returns a new HTTP request object, with the given method, url,
// request body.
func NewRequest(method, url string, reqBody *RequestBody) (*http.Request, error) {
	req, err := http.NewRequest(method, url, reqBody.Reader())
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}

	// For a regular request body, these fields are set automatically by
	// http.NewRequest.
	if req.Body != nil {
		if reqBody.ContentType() != "" {
			req.Header.Set("Content-Type", reqBody.ContentType())
		}
		req.ContentLength = reqBody.Len()
		req.GetBody = func() (io.ReadCloser, error) {
			return reqBody.Reader(), nil
		}
	}

	return req, nil
}

// NewRequest returns a new HTTP request object, with the given method, url,
// request body.
func FromConfig(c *configpb.HTTPRequest) (*http.Request, error) {
	req, err := NewRequest(c.GetMethod().String(), c.GetUrl(), NewRequestBody(c.GetData()...))
	if err != nil {
		return nil, err
	}
	for k, v := range c.GetHeader() {
		req.Header.Set(k, v)
	}
	return req, nil
}
