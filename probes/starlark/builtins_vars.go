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
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

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
