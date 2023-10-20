package proto

import (
	"github.com/cloudprober/cloudprober/internal/rds/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/rds/file/proto"
)

#TargetsConf: {
	// File that contains resources in either textproto or json format.
	// Example in textproto format:
	//
	// resource {
	//   name: "switch-xx-01"
	//   ip: "10.11.112.3"
	//   port: 8080
	//   labels {
	//     key: "device_type"
	//     value: "switch"
	//   }
	// }
	// resource {
	//   name: "switch-yy-01"
	//   ip: "10.16.110.12"
	//   port: 8080
	// }
	filePath?: string @protobuf(1,string,name=file_path)
	filter?: [...proto.#Filter] @protobuf(2,.cloudprober.rds.Filter)
	format?: proto_1.#ProviderConfig.#Format @protobuf(3,.cloudprober.rds.file.ProviderConfig.Format)

	// If specified, file will be re-read at the given interval.
	reEvalSec?: int32 @protobuf(4,int32,name=re_eval_sec)
}
