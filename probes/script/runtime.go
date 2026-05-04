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

// Notes on go.starlark.net threads:
//
// A starlark.Thread is a single, sequential evaluation context — it is NOT a
// goroutine. Concurrency in Starlark itself is non-existent; each thread runs
// one frame at a time. Threads are cheap to create (a struct with a name,
// frame stack, and a few hooks) so we make a fresh one for every probe run.
//
// Three thread features matter for this package:
//
//  1. thread.Cancel(reason): asks the interpreter to stop. The running script
//     is *not* interrupted mid-opcode — the interpreter checks the cancel
//     flag at every backwards branch and at every CALL/RETURN, then returns a
//     starlark.EvalError. This is why builtins that block on I/O (like our
//     http calls) MUST also honor the same cancellation: we attach the probe
//     ctx to outgoing http.Request objects so the I/O bails out promptly. If
//     a builtin blocks indefinitely with no ctx, thread.Cancel alone won't
//     unblock it.
//
//  2. thread.SetLocal/Local: a string-keyed scratch area on the thread,
//     invisible to Starlark code. We use it to thread the per-run
//     context.Context down to builtins (see ctxFromThread). Starlark has no
//     equivalent to Go's context plumbing, so this is the canonical pattern.
//
//  3. thread.Print: where Starlark's built-in print() goes. We route it to
//     the cloudprober logger so script-side debug lines land in the same
//     place as everything else.
//
// Globals:
//
// Module-level code (top-level `def`, assignments, etc.) runs once during
// starlark.ExecFile in NewRuntime. After ExecFile returns, the resulting
// StringDict is implicitly frozen for *mutation from new threads*: a fresh
// thread invoking probe() sees the globals as read-only. This means:
//
//   - You can't accumulate state in a module-level dict across probe runs.
//   - Helper functions defined at module level are safe to call concurrently
//     from multiple per-target threads (they have no shared mutable state).
//
// Cross-run state, when we add it in phase 2, will need an explicit `state`
// builtin backed by Go-side storage — not module globals.
package script

import (
	"context"
	"fmt"
	"net/http"

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

	// httpClient is the client used by the http builtin. Owned by Runtime
	// rather than reusing http.DefaultClient so configuration of one probe
	// can never leak into another (or into other in-process users of the
	// default client).
	httpClient *http.Client
}

// NewRuntime compiles the given Starlark source and verifies the entry point
// exists with the expected arity. Module-level code runs once here; runtime
// calls to Run cannot mutate the resulting globals (Starlark freezes them).
func NewRuntime(name, source, entryPoint string, l *logger.Logger) (*Runtime, error) {
	rt := &Runtime{
		name:       name,
		entryPoint: entryPoint,
		l:          l,
		httpClient: &http.Client{},
	}
	rt.predeclared = builtins()

	// One-shot thread used only to evaluate the file's top level. After
	// ExecFile returns we discard it; the resulting globals are reused.
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

// Thread-local keys. See top-of-file notes for the SetLocal/Local pattern.
const (
	threadCtxKey        = "cloudprober.ctx"
	threadHTTPClientKey = "cloudprober.httpClient"
)

// ctxFromThread returns the context stored on the Starlark thread.
func ctxFromThread(t *starlark.Thread) context.Context {
	if v := t.Local(threadCtxKey); v != nil {
		if ctx, ok := v.(context.Context); ok {
			return ctx
		}
	}
	return context.Background()
}

// httpClientFromThread returns the *http.Client owned by the Runtime that
// produced this thread. Falls back to http.DefaultClient only if the thread
// was constructed outside of Runtime.Run (shouldn't happen in practice; the
// fallback exists so unit tests can drive builtins directly).
func httpClientFromThread(t *starlark.Thread) *http.Client {
	if v := t.Local(threadHTTPClientKey); v != nil {
		if c, ok := v.(*http.Client); ok {
			return c
		}
	}
	return http.DefaultClient
}

// Run invokes the entry point with a target value. It returns nil on clean
// return, or an error on Starlark eval failure / assertion failure / ctx
// cancellation.
//
// Per call we build a brand-new starlark.Thread. Threads are cheap and giving
// each call its own avoids any chance of state leaking between runs (target
// X's failed assertion shouldn't poison target Y's thread.Cancel state).
// The global StringDict is shared and treated as read-only.
func (rt *Runtime) Run(ctx context.Context, ep endpoint.Endpoint) error {
	thread := &starlark.Thread{
		Name:  rt.name,
		Print: func(_ *starlark.Thread, msg string) { rt.l.Info(msg) },
	}
	// Stash ctx + httpClient so builtins can pull them back out via
	// ctxFromThread / httpClientFromThread. See top-of-file notes.
	thread.SetLocal(threadCtxKey, ctx)
	thread.SetLocal(threadHTTPClientKey, rt.httpClient)

	// Bridge ctx cancellation to thread.Cancel. The interpreter checks the
	// cancel flag at backward branches and call/return boundaries, so a
	// ctx-deadline triggers a starlark.EvalError on the next such boundary.
	// This handles cancellation of pure-Starlark loops; in-flight I/O is
	// cancelled separately via the ctx already attached to each request.
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
