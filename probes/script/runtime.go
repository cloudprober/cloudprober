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
	"context"
	"fmt"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Runtime owns a compiled Starlark program and its module-level globals.
// The compiled program is cached after Init; each Run call creates a fresh
// thread to invoke the entry point.
type Runtime struct {
	name        string
	entryPoint  string
	globals     starlark.StringDict
	predeclared starlark.StringDict
	l           *logger.Logger
}

// NewRuntime compiles the given Starlark source and verifies the entry point
// exists with the expected arity. Module-level code runs once here; runtime
// calls to Run cannot mutate the resulting globals (Starlark freezes them).
func NewRuntime(name, source, entryPoint string, l *logger.Logger) (*Runtime, error) {
	rt := &Runtime{
		name:       name,
		entryPoint: entryPoint,
		l:          l,
	}
	rt.predeclared = builtins()

	thread := &starlark.Thread{
		Name:  name + "-load",
		Print: func(_ *starlark.Thread, msg string) { l.Info(msg) },
	}
	globals, err := starlark.ExecFile(thread, name+".star", source, rt.predeclared)
	if err != nil {
		return nil, err
	}
	rt.globals = globals

	fn, ok := globals[entryPoint].(*starlark.Function)
	if !ok {
		return nil, fmt.Errorf("entry point %q not found or not a function", entryPoint)
	}
	if fn.NumParams() != 1 {
		return nil, fmt.Errorf("entry point %q must take exactly one argument (target), got %d", entryPoint, fn.NumParams())
	}
	return rt, nil
}

// Run invokes the entry point with a target value. It returns nil on clean
// return, or an error on Starlark eval failure / assertion failure / ctx
// cancellation.
func (rt *Runtime) Run(ctx context.Context, ep endpoint.Endpoint) error {
	thread := &starlark.Thread{
		Name:  rt.name,
		Print: func(_ *starlark.Thread, msg string) { rt.l.Info(msg) },
	}

	// Cancel the Starlark thread when ctx fires. Starlark interrupts at the
	// next opcode boundary and Call returns an error.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			thread.Cancel(ctx.Err().Error())
		case <-done:
		}
	}()

	fn := rt.globals[rt.entryPoint]
	target := targetValue(ep)
	_, err := starlark.Call(thread, fn, starlark.Tuple{target}, nil)
	return err
}

// targetValue builds the Starlark value passed to probe(target). Phase 1
// exposes .name and .port.
func targetValue(ep endpoint.Endpoint) starlark.Value {
	return starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"name": starlark.String(ep.Name),
		"port": starlark.MakeInt(ep.Port),
	})
}
