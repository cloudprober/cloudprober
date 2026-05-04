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

package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// builtins returns the predeclared globals available to every script.
// Phase 1 surface: http (get, post), assert (status).
func builtins() starlark.StringDict {
	return starlark.StringDict{
		"http":   httpModule(),
		"assert": assertModule(),
	}
}

// ----------------------------------------------------------------------------
// http module

func httpModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "http",
		Members: starlark.StringDict{
			"get":  starlark.NewBuiltin("http.get", httpGet),
			"post": starlark.NewBuiltin("http.post", httpPost),
		},
	}
}

func httpGet(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var url string
	var headers *starlark.Dict
	if err := starlark.UnpackArgs("http.get", args, kwargs,
		"url", &url,
		"headers?", &headers,
	); err != nil {
		return nil, err
	}
	return doHTTP(thread, "GET", url, headers, nil, nil)
}

func httpPost(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var url string
	var headers *starlark.Dict
	var body starlark.Value
	var jsonArg starlark.Value
	if err := starlark.UnpackArgs("http.post", args, kwargs,
		"url", &url,
		"headers?", &headers,
		"body?", &body,
		"json?", &jsonArg,
	); err != nil {
		return nil, err
	}
	return doHTTP(thread, "POST", url, headers, body, jsonArg)
}

func doHTTP(thread *starlark.Thread, method, url string, headers *starlark.Dict, body, jsonArg starlark.Value) (starlark.Value, error) {
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
		s, ok := starlark.AsString(body)
		if !ok {
			if b, ok := body.(starlark.Bytes); ok {
				reqBody = bytes.NewReader([]byte(b))
			} else {
				return nil, fmt.Errorf("http.%s: body must be string or bytes", strings.ToLower(method))
			}
		} else {
			reqBody = strings.NewReader(s)
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if headers != nil {
		for _, item := range headers.Items() {
			k, ok1 := starlark.AsString(item[0])
			v, ok2 := starlark.AsString(item[1])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("http.%s: headers keys and values must be strings", strings.ToLower(method))
			}
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	hdr := starlark.NewDict(len(resp.Header))
	for k, v := range resp.Header {
		_ = hdr.SetKey(starlark.String(k), starlark.String(strings.Join(v, ", ")))
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
	headers *starlark.Dict
	body    []byte
}

var _ starlark.Value = (*response)(nil)
var _ starlark.HasAttrs = (*response)(nil)

func (r *response) String() string        { return fmt.Sprintf("<response status=%d size=%d>", r.status, len(r.body)) }
func (r *response) Type() string          { return "Response" }
func (r *response) Freeze()               {}
func (r *response) Truth() starlark.Bool  { return starlark.Bool(r.status > 0) }
func (r *response) Hash() (uint32, error) { return 0, fmt.Errorf("Response is unhashable") }

func (r *response) Attr(name string) (starlark.Value, error) {
	switch name {
	case "status":
		return starlark.MakeInt(r.status), nil
	case "headers":
		return r.headers, nil
	case "body":
		return starlark.Bytes(r.body), nil
	case "json":
		return starlark.NewBuiltin("Response.json", r.jsonMethod).BindReceiver(r), nil
	}
	return nil, nil
}

func (r *response) AttrNames() []string {
	return []string{"status", "headers", "body", "json"}
}

func (r *response) jsonMethod(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 || len(kwargs) != 0 {
		return nil, fmt.Errorf("Response.json: takes no arguments")
	}
	var v interface{}
	if err := json.Unmarshal(r.body, &v); err != nil {
		return nil, fmt.Errorf("Response.json: %v", err)
	}
	return goToStarlark(v)
}

// ----------------------------------------------------------------------------
// assert module

func assertModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "assert",
		Members: starlark.StringDict{
			"status": starlark.NewBuiltin("assert.status", assertStatus),
		},
	}
}

func assertStatus(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resp starlark.Value
	var expected int
	if err := starlark.UnpackArgs("assert.status", args, kwargs,
		"response", &resp,
		"expected", &expected,
	); err != nil {
		return nil, err
	}
	r, ok := resp.(*response)
	if !ok {
		return nil, fmt.Errorf("assert.status: first argument must be a Response, got %s", resp.Type())
	}
	if r.status != expected {
		return nil, fmt.Errorf("assert.status: expected %d, got %d", expected, r.status)
	}
	return starlark.None, nil
}

// ----------------------------------------------------------------------------
// Starlark <-> Go conversion (minimal: enough for json marshal/unmarshal)

func starlarkToGo(v starlark.Value) (interface{}, error) {
	switch x := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(x), nil
	case starlark.Int:
		i, ok := x.Int64()
		if !ok {
			return float64(x.Float()), nil
		}
		return i, nil
	case starlark.Float:
		return float64(x), nil
	case starlark.String:
		return string(x), nil
	case *starlark.Dict:
		m := make(map[string]interface{}, x.Len())
		for _, item := range x.Items() {
			k, ok := starlark.AsString(item[0])
			if !ok {
				return nil, fmt.Errorf("dict key must be a string, got %s", item[0].Type())
			}
			gv, err := starlarkToGo(item[1])
			if err != nil {
				return nil, err
			}
			m[k] = gv
		}
		return m, nil
	case *starlark.List:
		out := make([]interface{}, 0, x.Len())
		it := x.Iterate()
		defer it.Done()
		var elem starlark.Value
		for it.Next(&elem) {
			gv, err := starlarkToGo(elem)
			if err != nil {
				return nil, err
			}
			out = append(out, gv)
		}
		return out, nil
	case starlark.Tuple:
		out := make([]interface{}, 0, x.Len())
		for i := 0; i < x.Len(); i++ {
			gv, err := starlarkToGo(x.Index(i))
			if err != nil {
				return nil, err
			}
			out = append(out, gv)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported Starlark type for json: %s", v.Type())
}

func goToStarlark(v interface{}) (starlark.Value, error) {
	switch x := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(x), nil
	case float64:
		if x == float64(int64(x)) {
			return starlark.MakeInt64(int64(x)), nil
		}
		return starlark.Float(x), nil
	case string:
		return starlark.String(x), nil
	case []interface{}:
		l := starlark.NewList(nil)
		for _, e := range x {
			sv, err := goToStarlark(e)
			if err != nil {
				return nil, err
			}
			if err := l.Append(sv); err != nil {
				return nil, err
			}
		}
		return l, nil
	case map[string]interface{}:
		d := starlark.NewDict(len(x))
		for k, vv := range x {
			sv, err := goToStarlark(vv)
			if err != nil {
				return nil, err
			}
			if err := d.SetKey(starlark.String(k), sv); err != nil {
				return nil, err
			}
		}
		return d, nil
	}
	return nil, fmt.Errorf("unsupported Go type for Starlark: %T", v)
}
