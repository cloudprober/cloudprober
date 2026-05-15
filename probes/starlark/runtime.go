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
// A starlarklib.Thread is a single, sequential evaluation context — it is NOT a
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
// starlarklib.ExecFile in newRuntime. After ExecFile returns, the resulting
// StringDict is implicitly frozen for *mutation from new threads*: a fresh
// thread invoking probe() sees the globals as read-only. This means:
//
//   - You can't accumulate state in a module-level dict across probe runs.
//   - Helper functions defined at module level are safe to call concurrently
//     from multiple per-target threads (they have no shared mutable state).
//
// Cross-run state, when we add it in phase 2, will need an explicit `state`
// builtin backed by Go-side storage — not module globals.
package starlark

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// runtime owns a compiled Starlark program and its module-level globals.
// The compiled program is cached after Init; each Run call creates a fresh
// thread to invoke the entry point.
type runtime struct {
	name        string
	entryPoint  string
	globals     starlarklib.StringDict
	predeclared starlarklib.StringDict
	l           *logger.Logger

	// httpClient is the client used by the http builtin. Owned by runtime
	// rather than reusing http.DefaultClient so configuration of one probe
	// can never leak into another (or into other in-process users of the
	// default client).
	httpClient *http.Client
}

// newRuntime compiles the given Starlark source and verifies the entry point
// exists with the expected arity. Module-level code runs once here and is
// bounded by ctx; runtime calls to Run cannot mutate the resulting globals
// (Starlark freezes them).
func newRuntime(ctx context.Context, name, source, entryPoint string, vars map[string]string, tlsCfg *tls.Config, l *logger.Logger) (*runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rt := &runtime{
		name:       name,
		entryPoint: entryPoint,
		l:          l,
		httpClient: newHTTPClient(tlsCfg),
	}
	rt.predeclared = builtins(vars)

	// One-shot thread used only to evaluate the file's top level. After
	// ExecFile returns we discard it; the resulting globals are reused.
	thread := &starlarklib.Thread{
		Name:  name + "-load",
		Print: func(_ *starlarklib.Thread, msg string) { l.Info(msg) },
	}
	thread.SetLocal(threadCtxKey, ctx)
	thread.SetLocal(threadHTTPClientKey, rt.httpClient)
	thread.SetLocal(threadLoggerKey, newLoggerHolder(l))
	// Module-level state.{get,set} writes go into a scratch bucket that's
	// discarded with the load thread. There's no target context at load time,
	// so persistence has nowhere to land.
	thread.SetLocal(threadStateKey, newStateBucket())
	// Module-level print_metric is a no-op: there's no probe run yet, so
	// nowhere to dispatch.
	thread.SetLocal(threadMetricEmitKey, metricEmitFn(func(string) {}))
	stopCancelBridge := cancelThreadOnContext(ctx, thread)
	defer stopCancelBridge()

	globals, err := starlarklib.ExecFile(thread, name+".star", source, rt.predeclared)
	if err != nil {
		return nil, err
	}
	rt.globals = globals

	fn, ok := globals[entryPoint].(*starlarklib.Function)
	if !ok {
		return nil, fmt.Errorf("entry point %q not found or not a function", entryPoint)
	}
	if fn.NumParams() != 1 {
		return nil, fmt.Errorf("entry point %q must take exactly one argument (target), got %d", entryPoint, fn.NumParams())
	}
	return rt, nil
}

// newHTTPClient returns the *http.Client the runtime uses for script-side
// http.{get,post} calls. With tlsCfg set, we clone DefaultTransport rather
// than starting from a zero-value Transport — Clone preserves the stdlib's
// dial/handshake/idle-conn timeouts, which a bare &http.Transport{} drops.
// Mutating DefaultTransport directly would leak into unrelated probes.
func newHTTPClient(tlsCfg *tls.Config) *http.Client {
	if tlsCfg == nil {
		return &http.Client{}
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = tlsCfg
	return &http.Client{Transport: t}
}

// Thread-local keys. See top-of-file notes for the SetLocal/Local pattern.
const (
	threadCtxKey        = "cloudprober.ctx"
	threadHTTPClientKey = "cloudprober.httpClient"
	threadLoggerKey     = "cloudprober.logger"
	threadStateKey      = "cloudprober.state"
	threadMetricEmitKey = "cloudprober.metricEmit"
)

// ctxFromThread returns the context stored on the Starlark thread.
func ctxFromThread(t *starlarklib.Thread) context.Context {
	if v := t.Local(threadCtxKey); v != nil {
		if ctx, ok := v.(context.Context); ok {
			return ctx
		}
	}
	return context.Background()
}

// httpClientFromThread returns the *http.Client owned by the runtime that
// produced this thread. Panics if the key is missing or the wrong type: every
// thread that runs script code is constructed by runtime, which sets the key.
// A miss means someone added a code path that builds a thread outside runtime
// losing the per-runtime client isolation we deliberately introduced — and we
// want that to fail loudly, not silently fall back to http.DefaultClient.
func httpClientFromThread(t *starlarklib.Thread) *http.Client {
	c, ok := t.Local(threadHTTPClientKey).(*http.Client)
	if !ok {
		panic("httpClientFromThread: thread missing httpClient local; constructed outside runtime?")
	}
	return c
}

// loggerHolder wraps a per-run *logger.Logger so script-side log.set_attr
// can swap in an enriched logger and have *both* further script-side log
// calls and the Go-side error-log line (run by runProbe after the script
// returns) see the new attributes. Without the holder, the script's
// mutation would only be visible inside its own thread.
type loggerHolder struct {
	base   *logger.Logger
	attrs  []slog.Attr
	keyPos map[string]int
	l      *logger.Logger
}

func newLoggerHolder(base *logger.Logger) *loggerHolder {
	return &loggerHolder{base: base, keyPos: map[string]int{}, l: base}
}

// SetAttr rebuilds l from base with the current attrs so repeated calls
// with the same key replace in place rather than append — preserves the
// script's set_attr call order in the log output, and avoids the duplicate
// fields Logger.WithAttributes would otherwise produce on repeat keys.
func (h *loggerHolder) SetAttr(key, value string) {
	attr := slog.String(key, value)
	if i, ok := h.keyPos[key]; ok {
		h.attrs[i] = attr
	} else {
		h.keyPos[key] = len(h.attrs)
		h.attrs = append(h.attrs, attr)
	}
	h.l = h.base.WithAttributes(h.attrs...)
}

// loggerFromThread returns the current *logger.Logger from the per-thread
// loggerHolder. Panics for the same reason httpClientFromThread does — every
// script thread is constructed by runtime/runProbe, which sets the key.
func loggerFromThread(t *starlarklib.Thread) *logger.Logger {
	return loggerHolderFromThread(t).l
}

// loggerHolderFromThread returns the per-run loggerHolder. log.set_attr uses
// it to install an enriched logger that persists for the rest of the run.
func loggerHolderFromThread(t *starlarklib.Thread) *loggerHolder {
	h, ok := t.Local(threadLoggerKey).(*loggerHolder)
	if !ok {
		panic("loggerHolderFromThread: thread missing logger local; constructed outside runtime?")
	}
	return h
}

// stateBucketFromThread returns the per-target state bucket stashed on the
// thread. Panics for the same reason httpClientFromThread does.
func stateBucketFromThread(t *starlarklib.Thread) *stateBucket {
	b, ok := t.Local(threadStateKey).(*stateBucket)
	if !ok {
		panic("stateBucketFromThread: thread missing state local; constructed outside runtime?")
	}
	return b
}

// metricEmitFromThread returns the per-run metric-emit callback stashed on
// the thread. Panics for the same reason httpClientFromThread does.
func metricEmitFromThread(t *starlarklib.Thread) metricEmitFn {
	f, ok := t.Local(threadMetricEmitKey).(metricEmitFn)
	if !ok {
		panic("metricEmitFromThread: thread missing metricEmit local; constructed outside runtime?")
	}
	return f
}

func cancelThreadOnContext(ctx context.Context, thread *starlarklib.Thread) func() {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			thread.Cancel(ctx.Err().Error())
		case <-done:
		}
	}()
	return func() {
		close(done)
	}
}

// Run invokes the entry point with a target value. It returns the final
// *logger.Logger (enriched by any log.set_attr calls in the script) so the
// caller can use it to log the script's error with the same attributes.
//
// l is the per-target logger, stashed on the thread so the log builtin can
// find it. Pass the per-target logger (not the probe-level one) so log lines
// from script code carry the target attribute attached by runProbe.
//
// bucket is the per-target state bucket. The caller (runProbe) owns its
// lifecycle by attaching it to runReq.TargetState; the scheduler frees it
// when the target is removed from discovery.
//
// metricEmit dispatches each print_metric line as the script runs (streaming,
// not buffered). runProbe wires this to a closure that parses the line and
// appends the resulting EventMetrics to result.payloadMetrics.
//
// Per call we build a brand-new starlarklib.Thread. Threads are cheap and giving
// each call its own avoids any chance of state leaking between runs (target
// X's failed assertion shouldn't poison target Y's thread.Cancel state).
// The global StringDict is shared and treated as read-only.
func (rt *runtime) Run(ctx context.Context, ep endpoint.Endpoint, l *logger.Logger, bucket *stateBucket, metricEmit metricEmitFn) (*logger.Logger, error) {
	lh := newLoggerHolder(l)
	thread := &starlarklib.Thread{
		Name:  rt.name,
		Print: func(_ *starlarklib.Thread, msg string) { lh.l.Info(msg) },
	}
	// Stash ctx + httpClient + logger + state bucket + metric-emit callback
	// so builtins can pull them back out via *FromThread helpers. See
	// top-of-file notes.
	thread.SetLocal(threadCtxKey, ctx)
	thread.SetLocal(threadHTTPClientKey, rt.httpClient)
	thread.SetLocal(threadLoggerKey, lh)
	thread.SetLocal(threadStateKey, bucket)
	thread.SetLocal(threadMetricEmitKey, metricEmit)

	// Bridge ctx cancellation to thread.Cancel. The interpreter checks the
	// cancel flag at backward branches and call/return boundaries, so a
	// ctx-deadline triggers a starlark.EvalError on the next such boundary.
	// This handles cancellation of pure-Starlark loops; in-flight I/O is
	// cancelled separately via the ctx already attached to each request.
	stopCancelBridge := cancelThreadOnContext(ctx, thread)
	defer stopCancelBridge()

	fn := rt.globals[rt.entryPoint]
	target := targetValue(ep)
	_, err := starlarklib.Call(thread, fn, starlarklib.Tuple{target}, nil)
	return lh.l, err
}

// targetValue builds the Starlark value passed to probe(target). It mirrors
// fields from endpoint.Endpoint:
//
//	target.name   — Endpoint.Name
//	target.port   — Endpoint.Port (0 if none)
//	target.ip     — Endpoint.IP.String() ("" if nil)
//	target.labels — Endpoint.Labels (frozen dict[str,str], empty if nil)
//
// Endpoint.LastUpdated is not currently exposed; add it if scripts need it.
func targetValue(ep endpoint.Endpoint) starlarklib.Value {
	ip := ""
	if ep.IP != nil {
		ip = ep.IP.String()
	}

	labels := starlarklib.NewDict(len(ep.Labels))
	for k, v := range ep.Labels {
		_ = labels.SetKey(starlarklib.String(k), starlarklib.String(v))
	}
	labels.Freeze()

	return starlarkstruct.FromStringDict(starlarkstruct.Default, starlarklib.StringDict{
		"name":   starlarklib.String(ep.Name),
		"port":   starlarklib.MakeInt(ep.Port),
		"ip":     starlarklib.String(ip),
		"labels": labels,
	})
}
