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

package tracing

import (
	"context"
	"net/url"

	"github.com/cloudprober/cloudprober/internal/tracing/otelsdk"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// newRedactingExporter wraps exp so that, for spans carrying an
// otelsdk.RedactQueryParamsAttrKey attribute, the named query parameters'
// values in the "url.full" attribute are replaced with "REDACTED" before
// spans are handed to exp. The marker attribute itself is stripped so it
// never reaches exp. Everything else about the span (and the actual
// outgoing request, which is untouched) is unaffected.
//
// This is a local stand-in for otelhttp.Transport's WithRedactedQueryParams,
// which doesn't exist yet
// (https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9643).
// Once cloudprober adopts an otelhttp release with that option, this
// exporter wrapper (and the marker attribute it reads) can be dropped in
// favor of configuring redaction directly on the Transport in
// probes/http/http.go.
func newRedactingExporter(exp sdktrace.SpanExporter) sdktrace.SpanExporter {
	return &redactingExporter{SpanExporter: exp}
}

type redactingExporter struct {
	sdktrace.SpanExporter
}

func (e *redactingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	out := make([]sdktrace.ReadOnlySpan, len(spans))
	for i, s := range spans {
		out[i] = maybeRedactSpan(s)
	}
	return e.SpanExporter.ExportSpans(ctx, out)
}

// maybeRedactSpan returns s unchanged unless it carries a
// otelsdk.RedactQueryParamsAttrKey attribute, in which case it returns a
// copy of s with the named query parameters' values redacted out of
// "url.full" and the marker attribute itself removed.
func maybeRedactSpan(s sdktrace.ReadOnlySpan) sdktrace.ReadOnlySpan {
	attrs := s.Attributes()

	var keys []string
	for _, a := range attrs {
		if a.Key == otelsdk.RedactQueryParamsAttrKey {
			keys = a.Value.AsStringSlice()
			break
		}
	}
	if len(keys) == 0 {
		return s
	}

	redacted := make([]attribute.KeyValue, 0, len(attrs)-1)
	for _, a := range attrs {
		if a.Key == otelsdk.RedactQueryParamsAttrKey {
			continue // Internal plumbing; never export it.
		}
		if a.Key == semconv.URLFullKey && len(keys) > 0 {
			if v, ok := redactURLQueryParams(a.Value.AsString(), keys); ok {
				redacted = append(redacted, semconv.URLFullKey.String(v))
				continue
			}
		}
		redacted = append(redacted, a)
	}
	return redactedSpan{ReadOnlySpan: s, attrs: redacted}
}

// redactURLQueryParams parses rawURL and replaces the values of any query
// parameters matching keys with "REDACTED". It returns the rewritten URL and
// true if at least one parameter was redacted; otherwise it returns rawURL
// unchanged and false. Non-redacted parameters are preserved, though
// url.Values.Encode() may reorder and canonically re-encode them.
func redactURLQueryParams(rawURL string, keys []string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.RawQuery == "" {
		return rawURL, false
	}

	q := u.Query()
	changed := false
	for _, k := range keys {
		if _, ok := q[k]; ok {
			q.Set(k, "REDACTED")
			changed = true
		}
	}
	if !changed {
		return rawURL, false
	}

	u.RawQuery = q.Encode()
	return u.String(), true
}

// redactedSpan wraps a ReadOnlySpan, overriding only Attributes().
type redactedSpan struct {
	sdktrace.ReadOnlySpan
	attrs []attribute.KeyValue
}

func (s redactedSpan) Attributes() []attribute.KeyValue {
	return s.attrs
}
