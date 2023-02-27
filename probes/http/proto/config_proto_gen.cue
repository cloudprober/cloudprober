package proto

import (
	"github.com/cloudprober/cloudprober/common/oauth/proto"
	proto_1 "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
)

// Next tag: 18
#ProbeConf: {
	#ProtocolType: {"HTTP", #enumValue: 0} |
		{"HTTPS", #enumValue: 1}

	#ProtocolType_value: {
		HTTP:  0
		HTTPS: 1
	}

	#Method: {"GET", #enumValue: 0} |
		{"POST", #enumValue: 1} |
		{"PUT", #enumValue: 2} |
		{"HEAD", #enumValue: 3} |
		{"DELETE", #enumValue: 4} |
		{"PATCH", #enumValue: 5} |
		{"OPTIONS", #enumValue: 6}

	#Method_value: {
		GET:     0
		POST:    1
		PUT:     2
		HEAD:    3
		DELETE:  4
		PATCH:   5
		OPTIONS: 6
	}

	#Header: {
		name?:  string @protobuf(1,string)
		value?: string @protobuf(2,string)
	}

	// Which HTTP protocol to use
	protocol?: #ProtocolType @protobuf(1,ProtocolType,"default=HTTP")

	// Relative URL (to append to all targets). Must begin with '/'
	relativeUrl?: string @protobuf(2,string,name=relative_url)

	// Port for HTTP requests. If not specfied, port is selected in the following
	// order:
	//  - If port is provided by the targets (e.g. kubernetes endpoint or
	//    service), that port is used.
	//  - 80 for HTTP and 443 for HTTPS.
	port?: int32 @protobuf(3,int32)

	// Whether to resolve the target before making the request. If set to false,
	// we hand over the target and relative_url directly to the golang's HTTP
	// module, Otherwise, we resolve the target first to an IP address and
	// make a request using that while passing target name as Host header.
	// By default we resolve first if it's a discovered resource, e.g., a k8s
	// endpoint.
	resolveFirst?: bool @protobuf(4,bool,name=resolve_first)

	// Export response (body) count as a metric
	exportResponseAsMetrics?: bool @protobuf(5,bool,name=export_response_as_metrics,"default=false")

	// HTTP request method
	method?: #Method @protobuf(7,Method,"default=GET")

	// HTTP request headers
	headers?: [...#Header] @protobuf(8,Header)

	// Request body.
	body?: string @protobuf(9,string)

	// Enable HTTP keep-alive. If set to true, underlying connection is reused
	// for further probes. Default is to close the connection after every request.
	keepAlive?: bool @protobuf(10,bool,name=keep_alive)

	// OAuth Config
	oauthConfig?: proto.#Config @protobuf(11,oauth.Config,name=oauth_config)

	// Disable HTTP2
	// Golang HTTP client automatically enables HTTP/2 if server supports it. This
	// option disables that behavior to enforce HTTP/1.1 for testing purpose.
	disableHttp2?: bool @protobuf(13,bool,name=disable_http2)

	// Disable TLS certificate validation. If set to true, any certificate
	// presented by the server for any host name will be accepted
	// Deprecation: This option is now subsumed by the tls_config below. To
	// disable cert validation use:
	// tls_config {
	//   disable_cert_validation: true
	// }
	disableCertValidation?: bool @protobuf(14,bool,name=disable_cert_validation)

	// TLS config
	tlsConfig?: proto_1.#TLSConfig @protobuf(15,tlsconfig.TLSConfig,name=tls_config)

	// Proxy URL, e.g. http://myproxy:3128
	proxyUrl?: string @protobuf(16,string,name=proxy_url)

	// Maximum idle connections to keep alive
	maxIdleConns?: int32 @protobuf(17,int32,name=max_idle_conns,"default=256")

	// The maximum amount of redirects the HTTP client will follow.
	// To disable redirects, use max_redirects: 0.
	maxRedirects?: int32 @protobuf(18,int32,name=max_redirects)

	// Interval between targets.
	intervalBetweenTargetsMsec?: int32 @protobuf(97,int32,name=interval_between_targets_msec,"default=10")

	// Requests per probe.
	// Number of HTTP requests per probe. Requests are executed concurrently and
	// each HTTP re contributes to probe results. For example, if you run two
	// requests per probe, "total" counter will be incremented by 2.
	requestsPerProbe?: int32 @protobuf(98,int32,name=requests_per_probe,"default=1")

	// How long to wait between two requests to the same target. Only relevant
	// if requests_per_probe is also configured.
	//
	// This value should be less than (interval - timeout) / requests_per_probe.
	// This is to ensure that all requests are executed within one probe interval
	// and all of them get sufficient time. For example, if probe interval is 2s,
	// timeout is 1s, and requests_per_probe is 10,  requests_interval_msec
	// should be less than 10ms.
	requestsIntervalMsec?: int32 @protobuf(99,int32,name=requests_interval_msec,"default=0")
}
