// Copyright 2017-2019 Google Inc.
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
package proto

// Next available tag = 10
#ServerConf: {
	port?: int32 @protobuf(1,int32,"default=3141")

	// tls_cert_file and tls_key_file field should be set for HTTPS.
	#ProtocolType: {"HTTP", #enumValue: 0} |
		{"HTTPS", #enumValue: 1}

	#ProtocolType_value: {
		HTTP:  0
		HTTPS: 1
	}
	protocol?: #ProtocolType @protobuf(6,ProtocolType,"default=HTTP")

	// Maximum duration for reading the entire request, including the body.
	readTimeoutMs?: int32 @protobuf(2,int32,name=read_timeout_ms,"default=10000") // default: 10s

	// Maximum duration before timing out writes of the response.
	writeTimeoutMs?: int32 @protobuf(3,int32,name=write_timeout_ms,"default=10000") // default: 10s

	// Maximum amount of time to wait for the next request when keep-alives are
	// enabled.
	idleTimeoutMs?: int32 @protobuf(4,int32,name=idle_timeout_ms,"default=60000") // default: 1m

	// Certificate file to use for HTTPS servers.
	tlsCertFile?: string @protobuf(7,string,name=tls_cert_file)

	// Private key file corresponding to the certificate above.
	tlsKeyFile?: string @protobuf(8,string,name=tls_key_file)

	// Disable HTTP/2 for HTTPS servers.
	disableHttp2?: bool @protobuf(9,bool,name=disable_http2)

	#PatternDataHandler: {
		// Response sizes to server, e.g. 1024.
		responseSize?: int32 @protobuf(1,int32,name=response_size)

		// Pattern is repeated to build the response, with "response_size mod
		// pattern_size" filled by '0' bytes.
		pattern?: string @protobuf(2,string,#"default="cloudprober""#)
	}

	// Pattern data handler returns pattern data at the url /data_<size_in_bytes>,
	// e.g. "/data_2048".
	patternDataHandler?: [...#PatternDataHandler] @protobuf(5,PatternDataHandler,name=pattern_data_handler)
}
