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

package otelsdk

import "go.opentelemetry.io/otel/attribute"

// RedactQueryParamsAttrKey is the span attribute key that tracing-capable
// probes use to carry, on each client span they create, the names of query
// parameters that should be redacted from that span's "url.full" attribute.
// The value is a string slice of parameter names.
//
// Probes set this attribute themselves at span-start time (e.g. via
// otelhttp.WithSpanOptions(trace.WithAttributes(...))), so redaction
// configuration travels with the span instead of living in shared, mutable
// state. internal/tracing's exporter wrapper reads this attribute, uses it
// to redact the named parameters' values out of "url.full", and strips the
// attribute itself so it never reaches a tracing backend.
//
// This is a local stand-in for redaction that otelhttp's Transport doesn't
// support yet
// (https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9643).
// Once that's available upstream, probes can configure it directly on the
// Transport, and this attribute (and the exporter wrapper that reads it)
// can be removed.
const RedactQueryParamsAttrKey attribute.Key = "cloudprober.redact_query_params_in_traces"
