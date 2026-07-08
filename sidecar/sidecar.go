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

// Package sidecar provides an SDK for writing cloudprober probe sidecars:
// external processes that implement probe types for cloudprober's
// EXTERNAL_GRPC probe. A new probe type is one struct and one Serve call:
//
//	var snowsql = sidecar.ProbeType[snowsqlConfig, *sql.DB]{
//	    New: func(ctx context.Context, t sidecar.Target, c snowsqlConfig) (*sql.DB, error) {
//	        return openSnowflake(ctx, t.Name, c.Warehouse)
//	    },
//	    Probe: func(ctx context.Context, t sidecar.Target, c snowsqlConfig, db *sql.DB) *sidecar.Result {
//	        start := time.Now()
//	        if err := db.QueryRowContext(ctx, c.Query).Scan(new(int)); err != nil {
//	            return sidecar.Fail(err)
//	        }
//	        return sidecar.OK(time.Since(start)).Metric("rows_scanned", 1)
//	    },
//	    Close: func(db *sql.DB) { db.Close() },
//	}
//
//	func main() {
//	    sidecar.Serve(
//	        sidecar.Listen("unix:///var/run/cloudprober/sidecar.sock"),
//	        sidecar.Register("snowsql", snowsql),
//	    )
//	}
//
// The SDK owns the gRPC server, health service, per-target session cache
// (keyed by the wire-level state_handle, which authors never see), and
// idle-TTL eviction.
package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Target mirrors cloudprober's targets/endpoint.Endpoint.
type Target struct {
	Name   string
	Labels map[string]string
	IP     string
	Port   int
}

// Result is the outcome of one probe run. Construct it with OK, Fail, or
// Internal; add extra metrics with Metric.
type Result struct {
	success           bool
	internal          bool
	invalidateSession bool
	err               error
	latency           time.Duration
	payload           []string
}

// OK reports a successful probe run with the given latency. Measure latency
// around the probe action itself so it excludes the cloudprober<->sidecar
// hop.
func OK(latency time.Duration) *Result {
	return &Result{success: true, latency: latency}
}

// Fail reports a target-level failure: the probe ran and the target is not
// healthy. This counts against the target's success ratio.
func Fail(err error) *Result {
	return &Result{err: err}
}

// Internal reports a sidecar/infra failure ("the probe broke", not "the
// target is down"). Cloudprober records it separately so it doesn't pollute
// the target's success ratio.
func Internal(err error) *Result {
	return &Result{internal: true, err: err}
}

// Metric attaches an additional metric to the result, with optional labels
// as key, value pairs:
//
//	sidecar.OK(latency).Metric("rows_scanned", 12).Metric("bytes", 4096, "phase", "scan")
//
// Metrics are encoded as cloudprober payload-format lines and parsed by
// cloudprober's metrics/payload parser. Numeric values are emitted as-is;
// string values starting with "dist:" or "map:" are passed through raw
// (that's how the parser spells distributions and maps); all other values
// (plain strings, bools, ...) are emitted as quoted strings. A dangling
// label key without a value gets an empty value.
func (r *Result) Metric(name string, value any, labels ...string) *Result {
	var sb strings.Builder
	sb.WriteString(name)
	if len(labels) > 0 {
		sb.WriteByte('{')
		for i := 0; i < len(labels); i += 2 {
			if i > 0 {
				sb.WriteByte(',')
			}
			val := ""
			if i+1 < len(labels) {
				val = labels[i+1]
			}
			fmt.Fprintf(&sb, "%s=%q", labels[i], val)
		}
		sb.WriteByte('}')
	}
	sb.WriteByte(' ')
	sb.WriteString(formatValue(value))
	r.payload = append(r.payload, sb.String())
	return r
}

// formatValue renders a metric value in payload format. The parser only
// understands numeric, "quoted string", "dist:...", and "map:..." values,
// so everything non-numeric is quoted (which also escapes newlines, keeping
// a value from injecting extra payload lines).
func formatValue(v any) string {
	switch val := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", val)
	case string:
		if strings.HasPrefix(val, "dist:") || strings.HasPrefix(val, "map:") {
			return val
		}
		return strconv.Quote(val)
	default:
		return strconv.Quote(fmt.Sprint(val))
	}
}

// InvalidateSession tells the SDK to discard (and Close) this target's
// session after the run, so the next probe cycle builds a fresh one via
// New. Use it when the probe detects that its cached session has gone bad
// in a way that won't self-heal (e.g. a dead raw connection or an expired
// auth session).
func (r *Result) InvalidateSession() *Result {
	r.invalidateSession = true
	return r
}

// ProbeType describes one probe type. C is the probe's typed config, decoded
// from the opaque config bytes cloudprober passes through; S is the
// per-target session (e.g. a connection pool).
//
// Stateless probe types set only Probe and use a throwaway session type
// (e.g. ProbeType[myConfig, any]); the session argument is then the zero
// value.
type ProbeType[C, S any] struct {
	// New builds the per-target session on first probe for a target (or when
	// the previous session was evicted). Optional.
	New func(ctx context.Context, t Target, c C) (S, error)

	// Probe runs one probe cycle for one target. Required.
	Probe func(ctx context.Context, t Target, c C, s S) *Result

	// Close releases a session when it's evicted (idle TTL). Optional.
	Close func(s S)
}

// handler is the type-erased form of ProbeType that the server dispatches
// on. Config decoding happens behind this seam so the server never sees
// typed configs or sessions.
type handler interface {
	runProbe(ctx context.Context, t Target, config []byte, session any) *Result
	newSession(ctx context.Context, t Target, config []byte) (any, error)
	closeSession(session any)
	validateConfig(config []byte) error
	stateful() bool
}

type typedHandler[C, S any] struct {
	pt ProbeType[C, S]
}

// configError marks config-decode failures, so the server can report them
// as internal errors (misconfiguration) while other newSession errors —
// from the author's New, which typically connects to the target — count as
// target-level failures.
type configError struct{ error }

// decodeConfig decodes config bytes into C with encoding/json. Empty config
// yields C's zero value. (protojson support for proto-typed configs is a
// planned follow-up.)
func (h typedHandler[C, S]) decodeConfig(config []byte) (C, error) {
	var c C
	if len(bytes.TrimSpace(config)) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(config, &c); err != nil {
		return c, configError{fmt.Errorf("decoding probe config: %v", err)}
	}
	return c, nil
}

func (h typedHandler[C, S]) validateConfig(config []byte) error {
	_, err := h.decodeConfig(config)
	return err
}

func (h typedHandler[C, S]) runProbe(ctx context.Context, t Target, config []byte, session any) *Result {
	c, err := h.decodeConfig(config)
	if err != nil {
		return Internal(err)
	}
	var s S
	if session != nil {
		s = session.(S)
	}
	return h.pt.Probe(ctx, t, c, s)
}

func (h typedHandler[C, S]) newSession(ctx context.Context, t Target, config []byte) (any, error) {
	c, err := h.decodeConfig(config)
	if err != nil {
		return nil, err
	}
	return h.pt.New(ctx, t, c)
}

func (h typedHandler[C, S]) closeSession(session any) {
	if h.pt.Close != nil {
		h.pt.Close(session.(S))
	}
}

func (h typedHandler[C, S]) stateful() bool {
	return h.pt.New != nil
}
