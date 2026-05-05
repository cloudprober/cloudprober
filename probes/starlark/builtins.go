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

	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// builtins returns the predeclared globals available to every script.
func builtins(vars map[string]string) starlarklib.StringDict {
	return starlarklib.StringDict{
		"http":   httpModule(),
		"assert": assertModule(),
		"vars":   varsModule(vars),
		"log":    logModule(),
	}
}

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

func httpGet(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var url string
	var headers *starlarklib.Dict
	if err := starlarklib.UnpackArgs("http.get", args, kwargs,
		"url", &url,
		"headers?", &headers,
	); err != nil {
		return nil, err
	}
	return doHTTP(thread, "GET", url, headers, nil, nil)
}

func httpPost(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var url string
	var headers *starlarklib.Dict
	var body starlarklib.Value
	var jsonArg starlarklib.Value
	if err := starlarklib.UnpackArgs("http.post", args, kwargs,
		"url", &url,
		"headers?", &headers,
		"body?", &body,
		"json?", &jsonArg,
	); err != nil {
		return nil, err
	}
	return doHTTP(thread, "POST", url, headers, body, jsonArg)
}

func doHTTP(thread *starlarklib.Thread, method, url string, headers *starlarklib.Dict, body, jsonArg starlarklib.Value) (starlarklib.Value, error) {
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

	resp, err := httpClientFromThread(thread).Do(req)
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

// ----------------------------------------------------------------------------
// vars module

func varsModule(vars map[string]string) *starlarkstruct.Module {
	get := func(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
		var name string
		var dflt starlarklib.Value = starlarklib.None
		if err := starlarklib.UnpackArgs("vars.get", args, kwargs,
			"name", &name,
			"default?", &dflt,
		); err != nil {
			return nil, err
		}
		if v, ok := vars[name]; ok {
			return starlarklib.String(v), nil
		}
		return dflt, nil
	}
	return &starlarkstruct.Module{
		Name: "vars",
		Members: starlarklib.StringDict{
			"get": starlarklib.NewBuiltin("vars.get", get),
		},
	}
}

// ----------------------------------------------------------------------------
// assert module

func assertModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "assert",
		Members: starlarklib.StringDict{
			"status": starlarklib.NewBuiltin("assert.status", assertStatus),
		},
	}
}

func assertStatus(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var resp starlarklib.Value
	var expected int
	if err := starlarklib.UnpackArgs("assert.status", args, kwargs,
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
	return starlarklib.None, nil
}

// ----------------------------------------------------------------------------
// log module
//
// log.{info,warn,error,debug}(msg) routes through the per-target logger
// stashed on the thread by runProbe (or the probe-level logger during
// load-time evaluation in NewRuntime). Single-string signature; scripts
// build composite messages with Starlark's % operator before calling.

func logModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "log",
		Members: starlarklib.StringDict{
			"info":  starlarklib.NewBuiltin("log.info", logAt("info")),
			"warn":  starlarklib.NewBuiltin("log.warn", logAt("warn")),
			"error": starlarklib.NewBuiltin("log.error", logAt("error")),
			"debug": starlarklib.NewBuiltin("log.debug", logAt("debug")),
		},
	}
}

func logAt(level string) func(*starlarklib.Thread, *starlarklib.Builtin, starlarklib.Tuple, []starlarklib.Tuple) (starlarklib.Value, error) {
	return func(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
		var msg string
		if err := starlarklib.UnpackArgs("log."+level, args, kwargs, "msg", &msg); err != nil {
			return nil, err
		}
		l := loggerFromThread(thread)
		switch level {
		case "info":
			l.Info(msg)
		case "warn":
			l.Warning(msg)
		case "error":
			l.Error(msg)
		case "debug":
			l.Debug(msg)
		}
		return starlarklib.None, nil
	}
}

// ----------------------------------------------------------------------------
// Starlark <-> Go conversion (minimal: enough for json marshal/unmarshal)

func starlarkToGo(v starlarklib.Value) (interface{}, error) {
	switch x := v.(type) {
	case starlarklib.NoneType:
		return nil, nil
	case starlarklib.Bool:
		return bool(x), nil
	case starlarklib.Int:
		i, ok := x.Int64()
		if !ok {
			return float64(x.Float()), nil
		}
		return i, nil
	case starlarklib.Float:
		return float64(x), nil
	case starlarklib.String:
		return string(x), nil
	case *starlarklib.Dict:
		m := make(map[string]interface{}, x.Len())
		for _, item := range x.Items() {
			k, ok := starlarklib.AsString(item[0])
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
	case *starlarklib.List:
		out := make([]interface{}, 0, x.Len())
		it := x.Iterate()
		defer it.Done()
		var elem starlarklib.Value
		for it.Next(&elem) {
			gv, err := starlarkToGo(elem)
			if err != nil {
				return nil, err
			}
			out = append(out, gv)
		}
		return out, nil
	case starlarklib.Tuple:
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

func goToStarlark(v interface{}) (starlarklib.Value, error) {
	switch x := v.(type) {
	case nil:
		return starlarklib.None, nil
	case bool:
		return starlarklib.Bool(x), nil
	case float64:
		if x == float64(int64(x)) {
			return starlarklib.MakeInt64(int64(x)), nil
		}
		return starlarklib.Float(x), nil
	case string:
		return starlarklib.String(x), nil
	case []interface{}:
		l := starlarklib.NewList(nil)
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
		d := starlarklib.NewDict(len(x))
		for k, vv := range x {
			sv, err := goToStarlark(vv)
			if err != nil {
				return nil, err
			}
			if err := d.SetKey(starlarklib.String(k), sv); err != nil {
				return nil, err
			}
		}
		return d, nil
	}
	return nil, fmt.Errorf("unsupported Go type for Starlark: %T", v)
}
