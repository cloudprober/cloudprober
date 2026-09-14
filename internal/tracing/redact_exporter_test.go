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
	"testing"

	"github.com/cloudprober/cloudprober/internal/tracing/otelsdk"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// exportSpan starts and immediately ends a span with the given attributes
// through a TracerProvider whose exporter is wrapped with
// newRedactingExporter, returning the single exported span's attributes.
func exportSpan(t *testing.T, attrs ...attribute.KeyValue) []attribute.KeyValue {
	t.Helper()

	inner := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(newRedactingExporter(inner)))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	_, span := tp.Tracer("test").Start(context.Background(), "GET /", trace.WithAttributes(attrs...))
	span.End()

	spans := inner.GetSpans()
	if !assert.Len(t, spans, 1, "expected exactly one exported span") {
		t.FailNow()
	}
	return spans[0].Attributes
}

func urlFullAttr(attrs []attribute.KeyValue) (string, bool) {
	for _, a := range attrs {
		if a.Key == semconv.URLFullKey {
			return a.Value.AsString(), true
		}
	}
	return "", false
}

// assertNoMarkerAttr fails if attrs still carries the internal
// otelsdk.RedactQueryParamsAttrKey marker; it should never reach an
// exporter.
func assertNoMarkerAttr(t *testing.T, attrs []attribute.KeyValue) {
	t.Helper()
	for _, a := range attrs {
		if a.Key == otelsdk.RedactQueryParamsAttrKey {
			t.Fatalf("marker attribute %q leaked into exported span", otelsdk.RedactQueryParamsAttrKey)
		}
	}
}

func TestRedactingExporterRedactsConfiguredParams(t *testing.T) {
	attrs := exportSpan(t,
		otelsdk.RedactQueryParamsAttrKey.StringSlice([]string{"token"}),
		semconv.URLFullKey.String("https://example.com/path?token=secret&other=1"),
	)

	got, ok := urlFullAttr(attrs)
	if assert.True(t, ok, "expected url.full attribute") {
		assert.Equal(t, "https://example.com/path?other=1&token=REDACTED", got)
	}
	assertNoMarkerAttr(t, attrs)
}

func TestRedactingExporterNoOpWithoutMarkerAttribute(t *testing.T) {
	const rawURL = "https://example.com/path?token=secret"

	attrs := exportSpan(t, semconv.URLFullKey.String(rawURL))

	got, ok := urlFullAttr(attrs)
	if assert.True(t, ok, "expected url.full attribute") {
		assert.Equal(t, rawURL, got)
	}
}

func TestRedactingExporterStripsMarkerEvenWhenParamNotPresent(t *testing.T) {
	const rawURL = "https://example.com/path?other=1"

	attrs := exportSpan(t,
		otelsdk.RedactQueryParamsAttrKey.StringSlice([]string{"not_in_url"}),
		semconv.URLFullKey.String(rawURL),
	)

	got, ok := urlFullAttr(attrs)
	if assert.True(t, ok, "expected url.full attribute") {
		assert.Equal(t, rawURL, got)
	}
	assertNoMarkerAttr(t, attrs)
}

func TestRedactURLQueryParams(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		keys   []string
		want   string
		wantOK bool
	}{
		{
			name:   "single param redacted",
			rawURL: "https://example.com/a?token=secret",
			keys:   []string{"token"},
			want:   "https://example.com/a?token=REDACTED",
			wantOK: true,
		},
		{
			name:   "other params preserved, possibly reordered",
			rawURL: "https://example.com/a?token=secret&other=1",
			keys:   []string{"token"},
			want:   "https://example.com/a?other=1&token=REDACTED",
			wantOK: true,
		},
		{
			name:   "key not present is a no-op",
			rawURL: "https://example.com/a?other=1",
			keys:   []string{"token"},
			want:   "https://example.com/a?other=1",
			wantOK: false,
		},
		{
			name:   "no query string is a no-op",
			rawURL: "https://example.com/a",
			keys:   []string{"token"},
			want:   "https://example.com/a",
			wantOK: false,
		},
		{
			name:   "no keys configured is a no-op",
			rawURL: "https://example.com/a?token=secret",
			keys:   nil,
			want:   "https://example.com/a?token=secret",
			wantOK: false,
		},
		{
			name:   "invalid URL is left unchanged",
			rawURL: "://not-a-url",
			keys:   []string{"token"},
			want:   "://not-a-url",
			wantOK: false,
		},
		{
			name:   "repeated key values are collapsed to one redacted value",
			rawURL: "https://example.com/a?token=one&token=two",
			keys:   []string{"token"},
			want:   "https://example.com/a?token=REDACTED",
			wantOK: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := redactURLQueryParams(tc.rawURL, tc.keys)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
