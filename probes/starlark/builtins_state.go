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

// The bucket lives on sched.RunProbeForTargetRequest.TargetState — created on
// the first run for a target, freed automatically when the target disappears
// from discovery (scheduler cancels the goroutine and drops the runReq).
// Single-goroutine-per-target is a scheduler invariant, so no internal lock.
//
// Values round-trip through starlarkToGo / goToStarlark, giving copy-on-get
// for free. Tuples preserve their identity (not flattened to lists) via the
// starlarkTuple named type. stateMaxKeys caps unique-key growth (key count,
// not byte size — a script that grows a single value unboundedly is not
// protected against). Big-ints that overflow int64 round-trip lossily
// through float64 (see the int64 case in goToStarlark below).

const stateMaxKeys = 1024

type stateBucket struct {
	values map[string]interface{}
}

func newStateBucket() *stateBucket {
	return &stateBucket{values: make(map[string]interface{})}
}

func stateModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "state",
		Members: starlarklib.StringDict{
			"get": starlarklib.NewBuiltin("state.get", stateGet),
			"set": starlarklib.NewBuiltin("state.set", stateSet),
		},
	}
}

func stateGet(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var key string
	var dflt starlarklib.Value = starlarklib.None
	if err := starlarklib.UnpackArgs("state.get", args, kwargs,
		"key", &key,
		"default?", &dflt,
	); err != nil {
		return nil, err
	}
	bucket := stateBucketFromThread(thread)
	v, ok := bucket.values[key]
	if !ok {
		return dflt, nil
	}
	return goToStarlark(v)
}

func stateSet(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var key string
	var value starlarklib.Value
	if err := starlarklib.UnpackArgs("state.set", args, kwargs,
		"key", &key,
		"value", &value,
	); err != nil {
		return nil, err
	}
	gv, err := starlarkToGo(value)
	if err != nil {
		return nil, fmt.Errorf("state.set: %v", err)
	}
	bucket := stateBucketFromThread(thread)
	if _, exists := bucket.values[key]; !exists && len(bucket.values) >= stateMaxKeys {
		return nil, fmt.Errorf("state.set: bucket exceeds max keys (%d)", stateMaxKeys)
	}
	bucket.values[key] = gv
	return starlarklib.None, nil
}
