package proto

import (
	"github.com/cloudprober/cloudprober/common/oauth/proto"
	proto_1 "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
)

// Generic HTTP request to notify on alert.
// You can use alert labels as HTTP request parameters. For example,
//  http_request {
//    method: GET
//    url: "https://api.pagerduty.com/v2/incidents"
//    json_params {
//      title: "type"
//      value: "incident"
//    }
//    ...
//  }
#HTTPRequest: {
	method?: string @protobuf(1,string)
	url?:    string @protobuf(2,string)
	headers?: {
		[string]: string
	} @protobuf(3,map[string]string)
	jsonParams?: {
		[string]: string
	} @protobuf(4,map[string]string,json_params)

	// Additional body to be sent with the request.
	body?: string @protobuf(5,string)

	// Proxy URL, e.g. http://myproxy:3128
	proxyUrl?: string @protobuf(6,string,name=proxy_url)

	// OAuth Config
	oauthConfig?: proto.#Config @protobuf(11,oauth.Config,name=oauth_config)

	// TLS config
	tlsConfig?: proto_1.#TLSConfig @protobuf(12,tlsconfig.TLSConfig,name=tls_config)
}

// Notify is not implemented yet.
#Notify: {
	cmd?:         string       @protobuf(1,string)
	email?:       string       @protobuf(2,string)
	httpRequest?: #HTTPRequest @protobuf(3,HTTPRequest,name=http_request)
}

#AlertConf: {
	// Name of the alert. Default is to use the probe name.
	name?: string @protobuf(1,string)

	// Thresholds for the alert.
	failureThreshold?: float32 @protobuf(2,float,name=failure_threshold)

	// Duration threshold in seconds. If duration_threshold_sec is set, alert
	// will be fired only if alert condition is true for
	// duration_threshold_sec.
	durationThresholdSec?: int32 @protobuf(3,int32,name=duration_threshold_sec)

	// How to notify in case of alert.
	notify?: #Notify @protobuf(4,Notify)
}
