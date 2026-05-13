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
	"fmt"

	starlarklib "go.starlark.net/starlark"
)

// builtins returns the predeclared globals available to every script.
// Each module lives in its own builtins_<name>.go file; this file is the
// registry plus the Starlark<->Go conversion shared by several of them.
func builtins(vars map[string]string) starlarklib.StringDict {
	return starlarklib.StringDict{
		"http":         httpModule(),
		"assert":       assertModule(),
		"vars":         varsModule(vars),
		"log":          logModule(),
		"state":        stateModule(),
		"print_metric": starlarklib.NewBuiltin("print_metric", printMetric),
	}
}

// ----------------------------------------------------------------------------
// Starlark <-> Go conversion (minimal: enough for json marshal/unmarshal)

// starlarkTuple distinguishes a Starlark Tuple from a List in the Go-side
// representation. It has []interface{} as the underlying type, so json.Marshal
// emits it as an array (the http(json=…) and Response.json() paths don't
// care). state.{set,get} reads the type back so a Tuple round-trips as a
// Tuple, preserving immutability and hashability.
type starlarkTuple []interface{}

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
		out := make(starlarkTuple, 0, x.Len())
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
	case int64:
		// starlarkToGo produces int64 for Starlark Int values that fit;
		// big-ints that overflow int64 fall through to its float64 branch,
		// so state round-trip is lossy for those (rare in practice). JSON
		// unmarshal never reaches this path — it emits float64 directly.
		return starlarklib.MakeInt64(x), nil
	case float64:
		if x == float64(int64(x)) {
			return starlarklib.MakeInt64(int64(x)), nil
		}
		return starlarklib.Float(x), nil
	case string:
		return starlarklib.String(x), nil
	case starlarkTuple:
		// Must precede the []interface{} case: a type switch matches in
		// order, and the underlying type would otherwise win.
		out := make(starlarklib.Tuple, 0, len(x))
		for _, e := range x {
			sv, err := goToStarlark(e)
			if err != nil {
				return nil, err
			}
			out = append(out, sv)
		}
		return out, nil
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
