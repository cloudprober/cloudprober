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
Package otel holds the OTLP exporter configuration shared by the
OpenTelemetry metrics surfacer (internal/surfacers/otel) and tracing
(internal/tracing). Both signals configure their exporters the same way, but
the OpenTelemetry SDK gives each signal its own, unrelated option type, so
what's shared here is the config interpretation, not the options themselves.
*/
package otel

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	configpb "github.com/cloudprober/cloudprober/internal/otel/proto"
)

// HTTPExporterTLSConfig validates an OTLP HTTP exporter's config and returns
// the tls.Config to use, or nil if the exporter doesn't configure TLS.
//
// endpoint_url is validated here because the SDK's WithEndpointURL only logs
// a parse failure and then falls back to the default endpoint; we'd rather
// fail at startup than silently export somewhere else.
func HTTPExporterTLSConfig(c *configpb.HTTPExporter) (*tls.Config, error) {
	if u := c.GetEndpointUrl(); u != "" {
		if _, err := url.Parse(u); err != nil {
			return nil, fmt.Errorf("otlp_http_exporter: invalid endpoint_url %q: %v", u, err)
		}
	}

	tlsConfig, err := tlsconfig.FromProto(c.GetTlsConfig())
	if err != nil {
		return nil, fmt.Errorf("otlp_http_exporter: failed to create tls config: %v", err)
	}
	return tlsConfig, nil
}

// GRPCExporterTLSConfig validates an OTLP gRPC exporter's config and returns
// the tls.Config to use, or nil if the exporter doesn't configure TLS (which
// includes the insecure case).
func GRPCExporterTLSConfig(c *configpb.GRPCExporter) (*tls.Config, error) {
	if c.GetInsecure() && c.GetTlsConfig() != nil {
		return nil, fmt.Errorf("otlp_grpc_exporter: insecure and tls_config are mutually exclusive")
	}
	tlsConfig, err := tlsconfig.FromProto(c.GetTlsConfig())
	if err != nil {
		return nil, fmt.Errorf("otlp_grpc_exporter: failed to create tls config: %v", err)
	}
	return tlsConfig, nil
}
