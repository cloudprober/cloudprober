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
// endpoint_url is validated here because the SDK's WithEndpointURL doesn't
// reject a bad URL -- it logs a parse failure and falls back to the default
// endpoint, and it derives the endpoint and transport security from the URL's
// host and scheme without checking that either is usable. We'd rather fail at
// startup than export nowhere. Note that a schemeless value like
// "otel.example.com:4318" (the shape otlp_grpc_exporter's endpoint takes)
// parses fine, as scheme "otel.example.com" with an empty host, so checking
// the parse error alone is not enough.
func HTTPExporterTLSConfig(c *configpb.HTTPExporter) (*tls.Config, error) {
	if endpointURL := c.GetEndpointUrl(); endpointURL != "" {
		u, err := url.Parse(endpointURL)
		if err != nil {
			return nil, fmt.Errorf("otlp_http_exporter: invalid endpoint_url %q: %v", endpointURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("otlp_http_exporter: endpoint_url %q must use the http or https scheme, got %q", endpointURL, u.Scheme)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("otlp_http_exporter: endpoint_url %q has no host", endpointURL)
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
