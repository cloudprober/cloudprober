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
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	starlarklib "go.starlark.net/starlark"
)

// ----------------------------------------------------------------------------
// starlarkToGo

func TestStarlarkToGo_Scalars(t *testing.T) {
	cases := []struct {
		name string
		in   starlarklib.Value
		want interface{}
	}{
		{"none", starlarklib.None, nil},
		{"bool true", starlarklib.Bool(true), true},
		{"bool false", starlarklib.Bool(false), false},
		{"int small", starlarklib.MakeInt(42), int64(42)},
		{"int negative", starlarklib.MakeInt(-7), int64(-7)},
		{"int zero", starlarklib.MakeInt(0), int64(0)},
		{"float", starlarklib.Float(3.14), float64(3.14)},
		{"float negative", starlarklib.Float(-2.5), float64(-2.5)},
		{"float zero", starlarklib.Float(0), float64(0)},
		{"string", starlarklib.String("hello"), "hello"},
		{"string empty", starlarklib.String(""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := starlarkToGo(tc.in)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestStarlarkToGo_BigInt covers the int64-overflow branch: Starlark ints are
// arbitrary precision, so values that don't fit in int64 fall back to float64.
func TestStarlarkToGo_BigInt(t *testing.T) {
	// math.MaxFloat64 / 1e10 ≈ 1.79e298, which is far beyond int64 max
	// (~9.22e18) but still finite. NumberToInt converts to a Starlark int.
	v, err := starlarklib.NumberToInt(starlarklib.Float(math.MaxFloat64 / 1e10))
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got, err := starlarkToGo(v)
	assert.NoError(t, err)
	_, isFloat := got.(float64)
	assert.True(t, isFloat, "expected float64 fallback for oversized int, got %T", got)
}

func TestStarlarkToGo_Dict(t *testing.T) {
	d := starlarklib.NewDict(3)
	_ = d.SetKey(starlarklib.String("name"), starlarklib.String("alice"))
	_ = d.SetKey(starlarklib.String("age"), starlarklib.MakeInt(30))
	_ = d.SetKey(starlarklib.String("admin"), starlarklib.Bool(true))

	got, err := starlarkToGo(d)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"name":  "alice",
		"age":   int64(30),
		"admin": true,
	}, got)
}

func TestStarlarkToGo_DictNonStringKey(t *testing.T) {
	d := starlarklib.NewDict(1)
	_ = d.SetKey(starlarklib.MakeInt(1), starlarklib.String("v"))

	_, err := starlarkToGo(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be a string")
}

func TestStarlarkToGo_DictEmpty(t *testing.T) {
	d := starlarklib.NewDict(0)
	got, err := starlarkToGo(d)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{}, got)
}

func TestStarlarkToGo_List(t *testing.T) {
	l := starlarklib.NewList([]starlarklib.Value{
		starlarklib.MakeInt(1),
		starlarklib.String("two"),
		starlarklib.Bool(false),
		starlarklib.None,
	})
	got, err := starlarkToGo(l)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{int64(1), "two", false, nil}, got)
}

func TestStarlarkToGo_ListEmpty(t *testing.T) {
	l := starlarklib.NewList(nil)
	got, err := starlarkToGo(l)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{}, got)
}

func TestStarlarkToGo_Tuple(t *testing.T) {
	tup := starlarklib.Tuple{
		starlarklib.String("a"),
		starlarklib.MakeInt(2),
	}
	got, err := starlarkToGo(tup)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{"a", int64(2)}, got)
}

func TestStarlarkToGo_Nested(t *testing.T) {
	// {"users": [{"name": "alice", "tags": ["a", "b"]}]}
	tags := starlarklib.NewList([]starlarklib.Value{
		starlarklib.String("a"),
		starlarklib.String("b"),
	})
	user := starlarklib.NewDict(2)
	_ = user.SetKey(starlarklib.String("name"), starlarklib.String("alice"))
	_ = user.SetKey(starlarklib.String("tags"), tags)
	users := starlarklib.NewList([]starlarklib.Value{user})
	root := starlarklib.NewDict(1)
	_ = root.SetKey(starlarklib.String("users"), users)

	got, err := starlarkToGo(root)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name": "alice",
				"tags": []interface{}{"a", "b"},
			},
		},
	}, got)
}

func TestStarlarkToGo_Unsupported(t *testing.T) {
	// A Starlark builtin is a Value that we don't (and shouldn't) marshal.
	fn := starlarklib.NewBuiltin("noop", func(_ *starlarklib.Thread, _ *starlarklib.Builtin, _ starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
		return starlarklib.None, nil
	})
	_, err := starlarkToGo(fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestStarlarkToGo_NestedError pins that errors from inner conversions
// (e.g. an unsupported value buried in a list) bubble up.
func TestStarlarkToGo_NestedError(t *testing.T) {
	bad := starlarklib.NewBuiltin("noop", func(_ *starlarklib.Thread, _ *starlarklib.Builtin, _ starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
		return starlarklib.None, nil
	})
	l := starlarklib.NewList([]starlarklib.Value{starlarklib.MakeInt(1), bad})
	_, err := starlarkToGo(l)
	assert.Error(t, err)

	d := starlarklib.NewDict(1)
	_ = d.SetKey(starlarklib.String("k"), bad)
	_, err = starlarkToGo(d)
	assert.Error(t, err)
}

// ----------------------------------------------------------------------------
// goToStarlark

func TestGoToStarlark_Scalars(t *testing.T) {
	cases := []struct {
		name     string
		in       interface{}
		wantType string
		wantStr  string
	}{
		{"nil", nil, "NoneType", "None"},
		{"bool true", true, "bool", "True"},
		{"bool false", false, "bool", "False"},
		{"int-valued float", float64(42), "int", "42"},
		{"negative int-valued float", float64(-5), "int", "-5"},
		{"true float", float64(3.14), "float", "3.14"},
		{"string", "hello", "string", `"hello"`},
		{"empty string", "", "string", `""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := goToStarlark(tc.in)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantType, got.Type())
			assert.Equal(t, tc.wantStr, got.String())
		})
	}
}

func TestGoToStarlark_Slice(t *testing.T) {
	got, err := goToStarlark([]interface{}{float64(1), "two", true, nil})
	assert.NoError(t, err)
	l, ok := got.(*starlarklib.List)
	if !ok {
		t.Fatalf("expected *List, got %T", got)
	}
	assert.Equal(t, 4, l.Len())
	assert.Equal(t, "1", l.Index(0).String())
	assert.Equal(t, `"two"`, l.Index(1).String())
	assert.Equal(t, "True", l.Index(2).String())
	assert.Equal(t, "None", l.Index(3).String())
}

func TestGoToStarlark_EmptySlice(t *testing.T) {
	got, err := goToStarlark([]interface{}{})
	assert.NoError(t, err)
	l, ok := got.(*starlarklib.List)
	if !ok {
		t.Fatalf("expected *List, got %T", got)
	}
	assert.Equal(t, 0, l.Len())
}

func TestGoToStarlark_Map(t *testing.T) {
	got, err := goToStarlark(map[string]interface{}{
		"name": "alice",
		"age":  float64(30),
	})
	assert.NoError(t, err)
	d, ok := got.(*starlarklib.Dict)
	if !ok {
		t.Fatalf("expected *Dict, got %T", got)
	}
	assert.Equal(t, 2, d.Len())
	v, ok, _ := d.Get(starlarklib.String("name"))
	assert.True(t, ok)
	assert.Equal(t, `"alice"`, v.String())
	v, ok, _ = d.Get(starlarklib.String("age"))
	assert.True(t, ok)
	assert.Equal(t, "30", v.String())
}

func TestGoToStarlark_EmptyMap(t *testing.T) {
	got, err := goToStarlark(map[string]interface{}{})
	assert.NoError(t, err)
	d, ok := got.(*starlarklib.Dict)
	if !ok {
		t.Fatalf("expected *Dict, got %T", got)
	}
	assert.Equal(t, 0, d.Len())
}

func TestGoToStarlark_Nested(t *testing.T) {
	// {"users": [{"name": "alice", "tags": ["a", "b"]}]}
	in := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name": "alice",
				"tags": []interface{}{"a", "b"},
			},
		},
	}
	got, err := goToStarlark(in)
	assert.NoError(t, err)
	d, ok := got.(*starlarklib.Dict)
	if !ok {
		t.Fatalf("expected *Dict, got %T", got)
	}
	usersV, ok, _ := d.Get(starlarklib.String("users"))
	assert.True(t, ok)
	users, ok := usersV.(*starlarklib.List)
	if !ok {
		t.Fatalf("expected list, got %T", usersV)
	}
	assert.Equal(t, 1, users.Len())
	user, ok := users.Index(0).(*starlarklib.Dict)
	if !ok {
		t.Fatalf("expected dict, got %T", users.Index(0))
	}
	tagsV, ok, _ := user.Get(starlarklib.String("tags"))
	assert.True(t, ok)
	tags, ok := tagsV.(*starlarklib.List)
	if !ok {
		t.Fatalf("expected list, got %T", tagsV)
	}
	assert.Equal(t, 2, tags.Len())
}

func TestGoToStarlark_Unsupported(t *testing.T) {
	// json.Unmarshal never produces an int — only float64 — but anything
	// caller-supplied must error rather than silently dropping.
	cases := []struct {
		name string
		in   interface{}
	}{
		{"int", int(1)},
		{"struct", struct{ X int }{1}},
		{"chan", make(chan int)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := goToStarlark(tc.in)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported")
		})
	}
}

// TestGoToStarlark_NestedError pins error propagation through containers.
func TestGoToStarlark_NestedError(t *testing.T) {
	_, err := goToStarlark([]interface{}{float64(1), int(2)}) // int unsupported
	assert.Error(t, err)

	_, err = goToStarlark(map[string]interface{}{"k": int(1)})
	assert.Error(t, err)
}

// ----------------------------------------------------------------------------
// JSON round-trip: this is the real-world shape — http.post(json=X) calls
// starlarkToGo(X) -> json.Marshal -> wire -> json.Unmarshal -> goToStarlark
// to produce r.json(). The Marshal+Unmarshal step is what normalizes int64
// into float64, which is why goToStarlark only accepts float64 numbers.

func TestRoundTrip_JSONviaStarlarkToGoToStarlark(t *testing.T) {
	// Build a dict that mirrors what http.post(json=...) would receive.
	src := starlarklib.NewDict(4)
	_ = src.SetKey(starlarklib.String("name"), starlarklib.String("alice"))
	_ = src.SetKey(starlarklib.String("active"), starlarklib.Bool(true))
	_ = src.SetKey(starlarklib.String("count"), starlarklib.MakeInt(7))
	tags := starlarklib.NewList([]starlarklib.Value{
		starlarklib.String("a"),
		starlarklib.String("b"),
	})
	_ = src.SetKey(starlarklib.String("tags"), tags)

	asGo, err := starlarkToGo(src)
	if err != nil {
		t.Fatalf("starlarkToGo: %v", err)
	}
	wire, err := json.Marshal(asGo)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded interface{}
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	asStarlark, err := goToStarlark(decoded)
	if err != nil {
		t.Fatalf("goToStarlark: %v", err)
	}

	d, ok := asStarlark.(*starlarklib.Dict)
	if !ok {
		t.Fatalf("expected *Dict, got %T", asStarlark)
	}
	v, _, _ := d.Get(starlarklib.String("name"))
	assert.Equal(t, `"alice"`, v.String())
	v, _, _ = d.Get(starlarklib.String("active"))
	assert.Equal(t, "True", v.String())
	// JSON normalizes the int through float64, but goToStarlark detects
	// integer-valued floats and emits a Starlark int, so 7 stays "7".
	v, _, _ = d.Get(starlarklib.String("count"))
	assert.Equal(t, "int", v.Type())
	assert.Equal(t, "7", v.String())
	tagsV, _, _ := d.Get(starlarklib.String("tags"))
	tagsL, ok := tagsV.(*starlarklib.List)
	if !ok {
		t.Fatalf("expected list, got %T", tagsV)
	}
	assert.Equal(t, 2, tagsL.Len())
	assert.Equal(t, `"a"`, tagsL.Index(0).String())
	assert.Equal(t, `"b"`, tagsL.Index(1).String())
}
