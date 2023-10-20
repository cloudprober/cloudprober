package proto

import (
	"github.com/cloudprober/cloudprober/internal/servers/http/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/servers/udp/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	proto_A "github.com/cloudprober/cloudprober/internal/servers/external/proto"
)

#ServerDef: {
	#Type: {"HTTP", #enumValue: 0} |
		{"UDP", #enumValue: 1} |
		{"GRPC", #enumValue: 2} |
		{"EXTERNAL", #enumValue: 3}

	#Type_value: {
		HTTP:     0
		UDP:      1
		GRPC:     2
		EXTERNAL: 3
	}
	type?: #Type @protobuf(1,Type)
	{} | {
		httpServer: proto.#ServerConf @protobuf(2,http.ServerConf,name=http_server)
	} | {
		udpServer: proto_1.#ServerConf @protobuf(3,udp.ServerConf,name=udp_server)
	} | {
		grpcServer: proto_5.#ServerConf @protobuf(4,grpc.ServerConf,name=grpc_server)
	} | {
		externalServer: proto_A.#ServerConf @protobuf(5,external.ServerConf,name=external_server)
	}
}
