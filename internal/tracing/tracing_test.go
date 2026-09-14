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

	tlsconfigpb "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
	"github.com/cloudprober/cloudprober/internal/tracing/otelsdk"
	configpb "github.com/cloudprober/cloudprober/internal/tracing/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/protobuf/proto"
)

func grpcExporterConfig() *configpb.TracingConfig_OtlpGrpcExporter {
	return &configpb.TracingConfig_OtlpGrpcExporter{
		OtlpGrpcExporter: &configpb.GRPCExporter{
			Endpoint: proto.String("localhost:4317"),
			Insecure: proto.Bool(true),
		},
	}
}

func TestGetExporterNoExporter(t *testing.T) {
	_, err := getExporter(context.Background(), &configpb.TracingConfig{})
	assert.Error(t, err, "expected error when no exporter is configured")
}

func TestGetExporterGRPCConflictingInsecureAndTLS(t *testing.T) {
	config := &configpb.TracingConfig{
		Exporter: &configpb.TracingConfig_OtlpGrpcExporter{
			OtlpGrpcExporter: &configpb.GRPCExporter{
				Endpoint: proto.String("localhost:4317"),
				Insecure: proto.Bool(true),
				TlsConfig: &tlsconfigpb.TLSConfig{
					CaCertFile: proto.String("/some/ca.pem"),
				},
			},
		},
	}
	_, err := getExporter(context.Background(), config)
	assert.Error(t, err, "expected error when insecure and tls_config are both set")
}

func TestInitSDKDisabled(t *testing.T) {
	t.Setenv(otelsdk.SDKDisabledEnvVar, "true")

	shutdown, err := Init(context.Background(), &configpb.TracingConfig{}, &logger.Logger{})
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestInitInvalidSamplingFraction(t *testing.T) {
	for _, sf := range []float64{-0.1, 1.1} {
		config := &configpb.TracingConfig{
			Exporter:         grpcExporterConfig(),
			SamplingFraction: proto.Float64(sf),
		}
		_, err := Init(context.Background(), config, &logger.Logger{})
		assert.Error(t, err, "expected error for sampling_fraction=%v", sf)
	}
}

func TestInitAndShutdown(t *testing.T) {
	// gRPC exporter construction does not dial immediately, so Init succeeds
	// without a live collector.
	config := &configpb.TracingConfig{
		Exporter: &configpb.TracingConfig_OtlpGrpcExporter{
			OtlpGrpcExporter: &configpb.GRPCExporter{
				Endpoint: proto.String("localhost:4317"),
				Insecure: proto.Bool(true),
			},
		},
		ResourceAttribute: []*configpb.TracingConfig_Attribute{
			{Key: proto.String("env"), Value: proto.String("test")},
		},
	}

	shutdown, err := Init(context.Background(), config, &logger.Logger{})
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	assert.NoError(t, shutdown(context.Background()))
}

func TestTraceResourceServiceName(t *testing.T) {
	for _, tc := range []struct {
		name        string
		otelEnv     string
		serviceName *string
		want        string
	}{
		{
			name: "default falls back to proto default",
			want: "cloudprober",
		},
		{
			name:    "OTEL_SERVICE_NAME takes effect when service_name is unset",
			otelEnv: "env-service",
			want:    "env-service",
		},
		{
			name:        "explicit service_name wins over OTEL_SERVICE_NAME",
			otelEnv:     "env-service",
			serviceName: proto.String("configured-service"),
			want:        "configured-service",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.otelEnv != "" {
				t.Setenv("OTEL_SERVICE_NAME", tc.otelEnv)
			}
			res, err := newResource(context.Background(), &configpb.TracingConfig{ServiceName: tc.serviceName})
			assert.NoError(t, err)
			v, ok := res.Set().Value(semconv.ServiceNameKey)
			assert.True(t, ok, "expected service.name to be set")
			assert.Equal(t, tc.want, v.AsString())
		})
	}
}

func TestInitDefaultPropagator(t *testing.T) {
	config := &configpb.TracingConfig{Exporter: grpcExporterConfig()}

	shutdown, err := Init(context.Background(), config, &logger.Logger{})
	assert.NoError(t, err)
	defer shutdown(context.Background())

	// Default propagators are tracecontext and baggage.
	fields := otel.GetTextMapPropagator().Fields()
	assert.Contains(t, fields, "traceparent")
	assert.Contains(t, fields, "baggage")
}

func TestInitInvalidPropagator(t *testing.T) {
	config := &configpb.TracingConfig{
		Exporter:   grpcExporterConfig(),
		Propagator: []string{"not-a-real-propagator"},
	}

	_, err := Init(context.Background(), config, &logger.Logger{})
	assert.Error(t, err, "expected error for unknown propagator name")
}
