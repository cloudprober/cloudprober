package proto

import (
	"github.com/cloudprober/cloudprober/internal/oauth/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
)

// Next tag: 21
#ProbeConf: {
	#Scheme: {"HTTP", #enumValue: 0} |
		{"HTTPS", #enumValue: 1}

	#Scheme_value: {
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

	// HTTP request scheme (Corresponding target label: "scheme"). If not set, we
	// use taget's 'scheme' label if present.
	// Note: protocol is deprecated, use scheme instead.
	{} | {
		protocol: #Scheme @protobuf(1,Scheme,"default=HTTP")
	} | {
		scheme: #Scheme @protobuf(21,Scheme,"default=HTTP")
	}

	// Relative URL (Corresponding target label: "path"). We construct the final
	// URL like this:
	// <scheme>://<host>:<port>/<relative_url>.
	//
	// Note that the relative_url should start with a '/'.
	relativeUrl?: string @protobuf(2,string,name=relative_url)

	// Port for HTTP requests (Corresponding target field: port)
	// Default is to use the scheme specific port, but if this field is not
	// set and discovered target has a port (e.g., k8s services, ingresses),
	// we use target's port.
	port?: int32 @protobuf(3,int32)

	// Whether to resolve the target before making the request. If set to true,
	// we resolve the target first to an IP address and make a request using
	// that while passing target name (or 'host' label if present) as Host
	// header.
	//
	// This behavior is automatic for discovered targets if they have an IP
	// address associated with them. Usually you don't need to worry about this
	// field and you can left it unspecified. We'll ty to do the right thing.
	resolveFirst?: bool @protobuf(4,bool,name=resolve_first)

	// Export response (body) count as a metric
	exportResponseAsMetrics?: bool @protobuf(5,bool,name=export_response_as_metrics,"default=false")

	// HTTP request method
	method?: #Method @protobuf(7,Method,"default=GET")

	// HTTP request headers
	// It is recommended to use "header" instead of "headers" for new configs.
	// header {
	//   key: "Authorization"
	//   value: "Bearer {{env "AUTH_TOKEN"}}"
	// }
	headers?: [...#Header] @protobuf(8,Header)
	header?: {
		[string]: string
	} @protobuf(20,map[string]string)

	// Request body. This field works similar to the curl's data flag. If there
	// are multiple "body" fields, we combine their values with a '&' in between.
	//
	// Also, we try to guess the content-type header based on the data:
	// 1) If data appears to be a valid json, we automatically set the
	//    content-type header to "application/json".
	// 2) If the final data string appears to be a valid query string, we
	//    set content-type to "application/x-www-form-urlencoded". Content type
	//    header can still be overridden using the header field above.
	// Example:
	//  body: "grant_type=client_credentials"
	//  body: "scope=transferMoney"
	//  body: "clientId=aweseomeClient"
	//  body: "clientSecret=noSecret"
	body?: [...string] @protobuf(9,string)

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

	// User agent. Default user agent is Go's default user agent.
	userAgent?: string @protobuf(19,string,name=user_agent)

	// Maximum idle connections to keep alive
	maxIdleConns?: int32 @protobuf(17,int32,name=max_idle_conns,"default=256")

	// The maximum amount of redirects the HTTP client will follow.
	// To disable redirects, use max_redirects: 0.
	maxRedirects?: int32 @protobuf(18,int32,name=max_redirects)

	#LatencyBreakdown: {"NO_BREAKDOWN", #enumValue: 0} |
		{"ALL_STAGES", #enumValue: 1} | {
			"DNS_LATENCY"// Exported as dns_latency
			#enumValue: 2
		} | {
			"CONNECT_LATENCY"// Exported as connect_latency
			#enumValue: 3
		} | {
			"TLS_HANDSHAKE_LATENCY"// Exported as tls_handshake_latency
			#enumValue: 4
		} | {
			"REQ_WRITE_LATENCY"// Exported as req_write_latency
			#enumValue: 5
		} | {
			"FIRST_BYTE_LATENCY"// Exported as first_byte_latency
			#enumValue: 6
		}

	#LatencyBreakdown_value: {
		NO_BREAKDOWN:          0
		ALL_STAGES:            1
		DNS_LATENCY:           2
		CONNECT_LATENCY:       3
		TLS_HANDSHAKE_LATENCY: 4
		REQ_WRITE_LATENCY:     5
		FIRST_BYTE_LATENCY:    6
	}

	// Add latency breakdown to probe results. This will add latency breakdown
	// by various stages of the request processing, e.g., DNS resolution, TCP
	// connection, TLS handshake, etc. You can select stages individually or
	// specify "ALL_STAGES" to get breakdown for all stages.
	//
	// Example:
	//   latency_breakdown: [ ALL_STAGES ]
	//   latency_breakdown: [ DNS_LATENCY, CONNECT_LATENCY, TLS_HANDSHAKE_LATENCY ]
	latencyBreakdown?: [...#LatencyBreakdown] @protobuf(22,LatencyBreakdown,name=latency_breakdown)

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
