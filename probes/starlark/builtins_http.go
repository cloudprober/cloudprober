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
			"get":    starlarklib.NewBuiltin("http.get", httpVerb("GET", false)),
			"post":   starlarklib.NewBuiltin("http.post", httpVerb("POST", true)),
			"put":    starlarklib.NewBuiltin("http.put", httpVerb("PUT", true)),
			"patch":  starlarklib.NewBuiltin("http.patch", httpVerb("PATCH", true)),
			"delete": starlarklib.NewBuiltin("http.delete", httpVerb("DELETE", true)),
		},
	}
}

// reqOpts holds per-call inputs to doHTTP. Fields default to their zero
// values when the corresponding kwarg is omitted.
type reqOpts struct {
	headers      *starlarklib.Dict
	body         starlarklib.Value
	jsonArg      starlarklib.Value
	maxRedirects *int
	keepAlive    bool
}

// optionalInt converts a Value bound by UnpackArgs (with "??" suffix) into a
// *int. nil result means the kwarg was omitted or None; otherwise the int
// value. Errors on any other type.
func optionalInt(v starlarklib.Value, name string) (*int, error) {
	if v == nil {
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

// httpVerb returns a builtin handler for the given HTTP method. When withBody
// is true, the handler also accepts "body" and "json" kwargs; GET omits them.
func httpVerb(method string, withBody bool) func(*starlarklib.Thread, *starlarklib.Builtin, starlarklib.Tuple, []starlarklib.Tuple) (starlarklib.Value, error) {
	name := "http." + strings.ToLower(method)
	return func(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
		var url string
		var opts reqOpts
		var maxRedirectsArg starlarklib.Value
		spec := []interface{}{
			"url", &url,
			"headers?", &opts.headers,
		}
		if withBody {
			spec = append(spec, "body?", &opts.body, "json?", &opts.jsonArg)
		}
		spec = append(spec, "max_redirects??", &maxRedirectsArg, "keep_alive??", &opts.keepAlive)
		if err := starlarklib.UnpackArgs(name, args, kwargs, spec...); err != nil {
			return nil, err
		}
		maxRedirects, err := optionalInt(maxRedirectsArg, name+": max_redirects")
		if err != nil {
			return nil, err
		}
		opts.maxRedirects = maxRedirects
		return doHTTP(thread, method, url, opts)
	}
}

func doHTTP(thread *starlarklib.Thread, method, url string, opts reqOpts) (starlarklib.Value, error) {
	var reqBody io.Reader
	contentType := ""
	switch {
	case opts.jsonArg != nil:
		raw, err := starlarkToGo(opts.jsonArg)
		if err != nil {
			return nil, fmt.Errorf("http.%s: encoding json arg: %v", strings.ToLower(method), err)
		}
		buf, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("http.%s: encoding json arg: %v", strings.ToLower(method), err)
		}
		reqBody = bytes.NewReader(buf)
		contentType = "application/json"
	case opts.body != nil:
		s, ok := starlarklib.AsString(opts.body)
		if !ok {
			if b, ok := opts.body.(starlarklib.Bytes); ok {
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
	if !opts.keepAlive {
		// req.Close: send Connection: close and don't return this conn
		// to the Transport's idle pool after the response.
		req.Close = true
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if opts.headers != nil {
		for _, item := range opts.headers.Items() {
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
		// Shallow-copy: the Transport pointer is shared, so connection
		// pooling is preserved across calls.
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
