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

package starlark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudprober/cloudprober/common/httpclient"
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ----------------------------------------------------------------------------
// http module

func httpModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "http",
		Members: starlarklib.StringDict{
			"get":  starlarklib.NewBuiltin("http.get", httpGet),
			"post": starlarklib.NewBuiltin("http.post", httpPost),
		},
	}
}

// reqOpts holds per-call knobs that mirror http probe config fields. nil
// pointers mean "unset" — fall back to the runtime's shared client default.
type reqOpts struct {
	maxRedirects *int
}

// optionalInt extracts a *int from a Starlark value bound by UnpackArgs.
// Returns nil for an omitted kwarg (v == nil) or an explicit None; errors
// for any other non-Int type. Used so optional int kwargs can distinguish
// "unset" from a real value without resorting to magic sentinels.
func optionalInt(v starlarklib.Value, name string) (*int, error) {
	if v == nil || v == starlarklib.None {
		return nil, nil
	}
	i, ok := v.(starlarklib.Int)
	if !ok {
		return nil, fmt.Errorf("%s: expected int, got %s", name, v.Type())
	}
	n, ok := i.Int64()
	if !ok {
		return nil, fmt.Errorf("%s: int does not fit in int64", name)
	}
	out := int(n)
	return &out, nil
}

func httpGet(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var url string
	var headers *starlarklib.Dict
	var maxRedirectsArg starlarklib.Value
	if err := starlarklib.UnpackArgs("http.get", args, kwargs,
		"url", &url,
		"headers?", &headers,
		"max_redirects?", &maxRedirectsArg,
	); err != nil {
		return nil, err
	}
	maxRedirects, err := optionalInt(maxRedirectsArg, "http.get: max_redirects")
	if err != nil {
		return nil, err
	}
	return doHTTP(thread, "GET", url, headers, nil, nil, reqOpts{maxRedirects: maxRedirects})
}

func httpPost(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var url string
	var headers *starlarklib.Dict
	var body starlarklib.Value
	var jsonArg starlarklib.Value
	var maxRedirectsArg starlarklib.Value
	if err := starlarklib.UnpackArgs("http.post", args, kwargs,
		"url", &url,
		"headers?", &headers,
		"body?", &body,
		"json?", &jsonArg,
		"max_redirects?", &maxRedirectsArg,
	); err != nil {
		return nil, err
	}
	maxRedirects, err := optionalInt(maxRedirectsArg, "http.post: max_redirects")
	if err != nil {
		return nil, err
	}
	return doHTTP(thread, "POST", url, headers, body, jsonArg, reqOpts{maxRedirects: maxRedirects})
}

func doHTTP(thread *starlarklib.Thread, method, url string, headers *starlarklib.Dict, body, jsonArg starlarklib.Value, opts reqOpts) (starlarklib.Value, error) {
	var reqBody io.Reader
	contentType := ""
	switch {
	case jsonArg != nil:
		raw, err := starlarkToGo(jsonArg)
		if err != nil {
			return nil, fmt.Errorf("http.%s: encoding json arg: %v", strings.ToLower(method), err)
		}
		buf, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("http.%s: encoding json arg: %v", strings.ToLower(method), err)
		}
		reqBody = bytes.NewReader(buf)
		contentType = "application/json"
	case body != nil:
		s, ok := starlarklib.AsString(body)
		if !ok {
			if b, ok := body.(starlarklib.Bytes); ok {
				reqBody = bytes.NewReader([]byte(b))
			} else {
				return nil, fmt.Errorf("http.%s: body must be string or bytes", strings.ToLower(method))
			}
		} else {
			reqBody = strings.NewReader(s)
		}
	}

	req, err := http.NewRequestWithContext(ctxFromThread(thread), method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if headers != nil {
		for _, item := range headers.Items() {
			k, ok1 := starlarklib.AsString(item[0])
			v, ok2 := starlarklib.AsString(item[1])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("http.%s: headers keys and values must be strings", strings.ToLower(method))
			}
			req.Header.Set(k, v)
		}
	}

	client := httpClientFromThread(thread)
	if opts.maxRedirects != nil {
		// Clone the client to apply per-call CheckRedirect; the Transport
		// pointer is shared, so connection pooling is preserved.
		clone := *client
		clone.CheckRedirect = httpclient.CheckRedirectFunc(*opts.maxRedirects)
		client = &clone
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Lossy: multi-valued headers (e.g. Set-Cookie) are joined with ", ".
	// This is wrong per RFC 7230 for Set-Cookie specifically, but it keeps
	// the Starlark-side type a flat dict[string,string]. Phase 2 should
	// expose the raw http.Header (multi-map) when scripts need it.
	hdr := starlarklib.NewDict(len(resp.Header))
	for k, v := range resp.Header {
		_ = hdr.SetKey(starlarklib.String(k), starlarklib.String(strings.Join(v, ", ")))
	}
	hdr.Freeze()

	return &response{
		status:  resp.StatusCode,
		headers: hdr,
		body:    respBody,
	}, nil
}

// ----------------------------------------------------------------------------
// Response value

type response struct {
	status  int
	headers *starlarklib.Dict
	body    []byte
}

var _ starlarklib.Value = (*response)(nil)
var _ starlarklib.HasAttrs = (*response)(nil)

func (r *response) String() string {
	return fmt.Sprintf("<response status=%d size=%d>", r.status, len(r.body))
}
func (r *response) Type() string            { return "Response" }
func (r *response) Freeze()                 {}
func (r *response) Truth() starlarklib.Bool { return starlarklib.Bool(r.status > 0) }
func (r *response) Hash() (uint32, error)   { return 0, fmt.Errorf("Response is unhashable") }

func (r *response) Attr(name string) (starlarklib.Value, error) {
	switch name {
	case "status":
		return starlarklib.MakeInt(r.status), nil
	case "headers":
		return r.headers, nil
	case "body":
		return starlarklib.Bytes(r.body), nil
	case "json":
		return starlarklib.NewBuiltin("Response.json", r.jsonMethod).BindReceiver(r), nil
	}
	return nil, nil
}

func (r *response) AttrNames() []string {
	return []string{"status", "headers", "body", "json"}
}

func (r *response) jsonMethod(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	if len(args) != 0 || len(kwargs) != 0 {
		return nil, fmt.Errorf("Response.json: takes no arguments")
	}
	var v interface{}
	if err := json.Unmarshal(r.body, &v); err != nil {
		return nil, fmt.Errorf("Response.json: %v", err)
	}
	return goToStarlark(v)
}
