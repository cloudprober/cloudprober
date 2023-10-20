package proto

import "github.com/cloudprober/cloudprober/internal/rds/proto"

// File provider config.
#ProviderConfig: {
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
	filePath?: [...string] @protobuf(1,string,name=file_path)

	#Format: {
		"UNSPECIFIED"// Determine format using file extension/
		#enumValue: 0
	} | {
		"TEXTPB"// Text proto format (.textpb).
		#enumValue: 1
	} | {
		"JSON"// JSON proto format (.json).
		#enumValue: 2
	}

	#Format_value: {
		UNSPECIFIED: 0
		TEXTPB:      1
		JSON:        2
	}
	format?: #Format @protobuf(2,Format)

	// If specified, file will be re-read at the given interval.
	reEvalSec?: int32 @protobuf(3,int32,name=re_eval_sec)

	// Whenever possible, we reload a file only if it has been modified since the
	// last load. If following option is set, mod time check is disabled.
	// Note that mod-time check doesn't work for GCS.
	disableModifiedTimeCheck?: bool @protobuf(4,bool,name=disable_modified_time_check)
}

#FileResources: {
	resource?: [...proto.#Resource] @protobuf(1,.cloudprober.rds.Resource)
}
