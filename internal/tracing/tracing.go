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

/*
Package tracing sets up distributed tracing for Cloudprober. It initializes a
global OpenTelemetry TracerProvider that exports spans to an OTLP collector
(over HTTP or gRPC).

All probes that support tracing are instrumented automatically. Whether a
given request is traced is decided by the global sampling fraction, and trace
context is propagated using the configured propagators.
*/
package tracing

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	configpb "github.com/cloudprober/cloudprober/internal/tracing/proto"
	"github.com/cloudprober/cloudprober/logger"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/credentials"
)

const sdkDisabledEnvVar = "OTEL_SDK_DISABLED"

// ShutdownFunc flushes any buffered spans and shuts down tracing resources.
type ShutdownFunc func(context.Context) error

// SDKDisabled reports whether OpenTelemetry SDK functionality is disabled by
// OTEL_SDK_DISABLED=true.
//
// cf: https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#general-sdk-configuration
func SDKDisabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(sdkDisabledEnvVar)), "true")
}

func getExporter(ctx context.Context, config *configpb.TracingConfig) (sdktrace.SpanExporter, error) {
	if config.GetOtlpHttpExporter() != nil {
		expConf := config.GetOtlpHttpExporter()

		var opts []otlptracehttp.Option

		if expConf.GetEndpointUrl() != "" {
			opts = append(opts, otlptracehttp.WithEndpointURL(expConf.GetEndpointUrl()))
		}

		if expConf.GetHttpHeader() != nil {
			opts = append(opts, otlptracehttp.WithHeaders(expConf.GetHttpHeader()))
		}

		if expConf.GetCompression() == configpb.Compression_GZIP {
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		}

		if expConf.GetTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			if err := tlsconfig.UpdateTLSConfig(tlsConfig, expConf.GetTlsConfig()); err != nil {
				return nil, fmt.Errorf("failed to create tls config: %v", err)
			}
			opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
		}

		return otlptracehttp.New(ctx, opts...)
	}

	if config.GetOtlpGrpcExporter() != nil {
		expConf := config.GetOtlpGrpcExporter()

		var opts []otlptracegrpc.Option

		if expConf.GetEndpoint() != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(expConf.GetEndpoint()))
		}

		if expConf.GetHttpHeader() != nil {
			opts = append(opts, otlptracegrpc.WithHeaders(expConf.GetHttpHeader()))
		}

		if expConf.GetCompression() == configpb.Compression_GZIP {
			opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
		}

		if expConf.GetInsecure() && expConf.GetTlsConfig() != nil {
			return nil, fmt.Errorf("otlp_grpc_exporter: insecure and tls_config are mutually exclusive")
		}

		if expConf.GetInsecure() {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		if expConf.GetTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			if err := tlsconfig.UpdateTLSConfig(tlsConfig, expConf.GetTlsConfig()); err != nil {
				return nil, fmt.Errorf("failed to create tls config: %v", err)
			}
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
		}

		return otlptracegrpc.New(ctx, opts...)
	}

	return nil, fmt.Errorf("no OTLP exporter configured in tracing config")
}

// Init sets up a global OpenTelemetry TracerProvider and propagator based on
// the provided config. It returns a shutdown function so the caller can flush
// and shut tracing down on exit. Init should be called once, before probes are
// initialized.
func Init(ctx context.Context, config *configpb.TracingConfig, l *logger.Logger) (ShutdownFunc, error) {
	if SDKDisabled() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
		l.Infof("OpenTelemetry tracing disabled by %s", sdkDisabledEnvVar)
		return func(context.Context) error { return nil }, nil
	}

	// Resource attributes are applied after WithFromEnv so that explicitly
	// configured values (including service_name) take precedence over
	// OTEL_RESOURCE_ATTRIBUTES/OTEL_SERVICE_NAME, matching the OTEL surfacer.
	attrKVs := []attribute.KeyValue{attribute.String("service.name", config.GetServiceName())}
	for _, attr := range config.GetResourceAttribute() {
		attrKVs = append(attrKVs, attribute.String(attr.GetKey(), attr.GetValue()))
	}
	res, err := resource.New(ctx, resource.WithHost(), resource.WithFromEnv(), resource.WithAttributes(attrKVs...))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace resource: %v", err)
	}

	// Build the propagator from the configured names. When none are configured,
	// fall back to OTEL_PROPAGATORS, defaulting to tracecontext and baggage.
	prop := autoprop.NewTextMapPropagator()
	if names := config.GetPropagator(); len(names) > 0 {
		prop, err = autoprop.TextMapPropagator(names...)
		if err != nil {
			return nil, fmt.Errorf("invalid propagator config: %v", err)
		}
	}

	exp, err := getExporter(ctx, config)
	if err != nil {
		return nil, err
	}

	// Cloudprober is the root of the trace, so the root sampler decides
	// whether a request is traced based on the configured fraction.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.GetSamplingFraction()))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(prop)

	l.Infof("Initialized opentelemetry tracing with config: %s", config.String())
	return tp.Shutdown, nil
}
