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
)

// print_metric(line) hands the line to a metrics/payload Parser via a
// per-run callback installed on the thread by runProbe. Streaming, not
// buffered: each call dispatches its EventMetrics immediately, matching
// external probe behavior. Metrics emitted before a script error survive.
//
// Line format is exactly what the parser already accepts (cloudprober's
// payload syntax), so distributions, GAUGE/CUMULATIVE kind, in-cloudprober
// aggregation, label syntax, and JSON / header metrics all carry over with
// zero re-projection in Starlark:
//
//	print_metric("items_in_cart 5")
//	print_metric('items_in_cart{user="alice"} 5')
//	print_metric("checkout_latency_ms 234.5")  # routed to dist if configured
//	print_metric("dist:sum:899|count:221|lb:-Inf,0.5,2|bc:34,54,121")

// metricEmitFn dispatches one payload-format line. runProbe builds a closure
// that parses the line and appends the resulting EventMetrics to
// result.payloadMetrics; module-load installs a no-op.
type metricEmitFn func(line string)

func printMetric(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var line string
	if err := starlarklib.UnpackArgs("print_metric", args, kwargs, "line", &line); err != nil {
		return nil, err
	}
	metricEmitFromThread(thread)(line)
	return starlarklib.None, nil
}
