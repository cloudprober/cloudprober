package proto

import "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"

#Compression: {"NONE", #enumValue: 0} |
	{"GZIP", #enumValue: 1}

#Compression_value: {
	NONE: 0
	GZIP: 1
}

#HTTPExporter: {
	// If no URL is provided, OpenTelemetry SDK will use the environment variable
	// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT or OTEL_EXPORTER_OTLP_ENDPOINT in that
	// preference order.
	endpointUrl?: string           @protobuf(1,string,name=endpoint_url)
	tlsConfig?:   proto.#TLSConfig @protobuf(2,tlsconfig.TLSConfig,name=tls_config)

	// HTTP request headers. These can also be set using environment variables.
	httpHeader?: {
		[string]: string
	} @protobuf(3,map[string]string,http_header)

	// Compression algorithm to use for HTTP requests.
	compression?: #Compression @protobuf(4,Compression)
}

#GRPCExporter: {
	// If no URL is provided, OpenTelemetry SDK will use the environment variable
	// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT or OTEL_EXPORTER_OTLP_ENDPOINT in that
	// preference order.
	endpoint?:  string           @protobuf(1,string)
	tlsConfig?: proto.#TLSConfig @protobuf(2,tlsconfig.TLSConfig,name=tls_config)

	// HTTP request headers. These can also be set using environment variables.
	httpHeader?: {
		[string]: string
	} @protobuf(3,map[string]string,http_header)

	// Compression algorithm to use for gRPC requests.
	compression?: #Compression @protobuf(4,Compression)

	// Whether to use insecure gRPC connection.
	insecure?: bool @protobuf(5,bool)
}

#SurfacerConf: {
	{} | {
		// OTLP HTTP exporter.
		otlpHttpExporter: #HTTPExporter @protobuf(1,HTTPExporter,name=otlp_http_exporter)
	} | {
		// OTLP gRPC exporter.
		otlpGrpcExporter: #GRPCExporter @protobuf(2,GRPCExporter,name=otlp_grpc_exporter)
	}
	exportIntervalSec?: int32 @protobuf(3,int32,name=export_interval_sec,"default=10")
}
