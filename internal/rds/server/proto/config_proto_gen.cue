package proto

import (
	"github.com/cloudprober/cloudprober/internal/rds/file/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/rds/gcp/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
)

#ServerConf: {
	// List of providers that server supports.
	provider?: [...#Provider] @protobuf(1,Provider)
}

#Provider: {
	// Provider identifier, e.g. "gcp". Server routes incoming requests to various
	// providers based on this id.
	id?: string @protobuf(1,string)
	{} | {
		fileConfig: proto.#ProviderConfig @protobuf(4,file.ProviderConfig,name=file_config)
	} | {
		gcpConfig: proto_1.#ProviderConfig @protobuf(2,gcp.ProviderConfig,name=gcp_config)
	} | {
		kubernetesConfig: proto_5.#ProviderConfig @protobuf(3,kubernetes.ProviderConfig,name=kubernetes_config)
	}
}
