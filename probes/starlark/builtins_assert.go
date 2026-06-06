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
	"go.starlark.net/starlarkstruct"
)

func assertModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "assert",
		Members: starlarklib.StringDict{
			"http_status": starlarklib.NewBuiltin("assert.http_status", assertHTTPStatus),
		},
	}
}

func assertHTTPStatus(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var resp starlarklib.Value
	var expected int
	if err := starlarklib.UnpackArgs("assert.http_status", args, kwargs,
		"response", &resp,
		"expected", &expected,
	); err != nil {
		return nil, err
	}
	r, ok := resp.(*response)
	if !ok {
		return nil, fmt.Errorf("assert.http_status: first argument must be a Response, got %s", resp.Type())
	}
	if r.status != expected {
		return nil, fmt.Errorf("assert.http_status: expected %d, got %d", expected, r.status)
	}
	return starlarklib.None, nil
}
