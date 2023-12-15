package proto

import (
	"github.com/cloudprober/cloudprober/internal/rds/client/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/rds/proto"
	proto_5 "github.com/cloudprober/cloudprober/targets/gce/proto"
	proto_A "github.com/cloudprober/cloudprober/targets/file/proto"
	proto_8 "github.com/cloudprober/cloudprober/targets/lameduck/proto"
)

#RDSTargets: {
	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	rdsServerOptions?: proto.#ClientConf.#ServerOptions @protobuf(1,rds.ClientConf.ServerOptions,name=rds_server_options)

	// Resource path specifies the resources to return. Resources paths have the
	// following format:
	// <resource_provider>://<resource_type>/<additional_params>
	//
	// Examples:
	// For GCE instances in projectA: "gcp://gce_instances/<projectA>"
	// Kubernetes Pods : "k8s://pods"
	resourcePath?: string @protobuf(2,string,name=resource_path)

	// Filters to filter resources by.
	filter?: [...proto_1.#Filter] @protobuf(3,rds.Filter)

	// IP config to specify the IP address to pick for a resource.
	ipConfig?: proto_1.#IPConfig @protobuf(4,rds.IPConfig,name=ip_config)
}

#K8sTargets: {
	// Targets namespace. If this field is unset, we select resources from all
	// namespaces.
	namespace?: string @protobuf(1,string)

	// labelSelector uses the same format as kubernetes API calls.
	// Example:
	//   labelSelector: "k8s-app"       # label k8s-app exists
	//   labelSelector: "role=frontend" # label role=frontend
	//   labelSelector: "!canary"       # canary label doesn't exist
	labelSelector?: [...string] @protobuf(2,string)
	// Which resources to target. If value is not empty (""), we use it as a
	// regex for resource names.
	// Example:
	//   services: ""             // All services.
	//   endpoints: ".*-service"  // Endpoints ending with "service".
	{} | {
		services: string @protobuf(3,string)
	} | {
		endpoints: string @protobuf(4,string)
	} | {
		ingresses: string @protobuf(5,string)
	} | {
		pods: string @protobuf(6,string)
	}

	// portFilter can be used to filter resources by port name. This is useful
	// for resources like endpoints and services, where each resource may have
	// multiple ports, and we may hit just a subset of those ports. portFilter
	// takes a regex -- we apply it on port names if port name is available,
	// otherwise we apply it port numbers.
	// Example: ".*-dns", "metrics", ".*-service", etc.
	portFilter?: string @protobuf(10,string)

	// How often to re-check k8s API servers. Note this field will be irrelevant
	// when (and if) we move to the watch API. Default is 30s.
	reEvalSec?:        int32                            @protobuf(19,int32,name=re_eval_sec)
	rdsServerOptions?: proto.#ClientConf.#ServerOptions @protobuf(20,rds.ClientConf.ServerOptions,name=rds_server_options)
}

#Endpoint: {
	// Endpoint name. Metrics for a target are identified by a combination of
	// endpoint name and port name, if specified.
	name?: string @protobuf(1,string)

	// Optional IP address. If not specified, endpoint name is DNS resolved.
	ip?: string @protobuf(2,string)

	// Endpoint port. If specified, this port will be used by the port-based
	// probes (e.g.  TCP, HTTP), if probe's configuration doesn't specify a port.
	port?: int32 @protobuf(3,int32)

	// HTTP probe URL. If provided, this field is used by the HTTP probe, if
	// probe configuration itself doesn't specify URL fields.
	url?: string @protobuf(4,string)

	// Endpoint labels. These labels can be exported as metrics labels using the
	// `additional_label` field in the probe configuration.
	labels?: {
		[string]: string
	} @protobuf(5,map[string]string)
}

#TargetsDef: {
	{} | {
		// Static host names, for example:
		// host_name: "www.google.com,8.8.8.8,en.wikipedia.org"
		hostNames: string @protobuf(1,string,name=host_names)
	} | {
		// Shared targets are accessed through their names.
		// Example:
		// shared_targets {
		//   name:"backend-vms"
		//   targets {
		//     rds_targets {
		//       ..
		//     }
		//   }
		// }
		//
		// probe {
		//   targets {
		//     shared_targets: "backend-vms"
		//   }
		// }
		sharedTargets: string @protobuf(5,string,name=shared_targets)
	} | {
		// GCE targets: instances and forwarding_rules, for example:
		// gce_targets {
		//   instances {}
		// }
		gceTargets: proto_5.#TargetsConf @protobuf(2,gce.TargetsConf,name=gce_targets)
	} | {
		// ResourceDiscovery service based targets.
		// Example:
		// rds_targets {
		//   resource_path: "gcp://gce_instances/{{.project}}"
		//   filter {
		//     key: "name"
		//     value: ".*backend.*"
		//   }
		// }
		rdsTargets: #RDSTargets @protobuf(3,RDSTargets,name=rds_targets)
	} | {
		// File based targets.
		// Example:
		// file_targets {
		//   file_path: "/var/run/cloudprober/vips.textpb"
		// }
		fileTargets: proto_A.#TargetsConf @protobuf(4,file.TargetsConf,name=file_targets)
	} | {
		// K8s targets.
		// Note: k8s targets are still in the experimental phase. Their config API
		// may change in the future.
		// Example:
		// k8s {
		//   namespace: "qa"
		//   labelSelector: "k8s-app"
		//   services: ""
		// }
		k8s: #K8sTargets @protobuf(6,K8sTargets)
	} | {
		// Empty targets to meet the probe definition requirement where there are
		// actually no targets, for example in case of some external probes.
		dummyTargets: #DummyTargets @protobuf(20,DummyTargets,name=dummy_targets)
	}

	// Static endpoints. These endpoints are merged with the resources returned
	// by the targets type above.
	// Example:
	//   endpoint {
	//     name: "service-gtwy-1"
	//     ip: "10.1.18.121"
	//     port: 8080
	//     labels {
	//       key: "service"
	//       value: "products-service"
	//     }
	//   }
	//   endpoint {
	//     name: "frontend-url1"
	//     url: "https://frontend.example.com/url1"
	//   }
	endpoint?: [...#Endpoint] @protobuf(23,Endpoint)

	// Regex to apply on the targets.
	regex?: string @protobuf(21,string)

	// Exclude lameducks. Lameduck targets can be set through RTC (realtime
	// configurator) service. This functionality works only if lame_duck_options
	// are specified.
	excludeLameducks?: bool @protobuf(22,bool,name=exclude_lameducks,default)
}

// DummyTargets represent empty targets, which are useful for external
// probes that do not have any "proper" targets.  Such as ilbprober.
#DummyTargets: {
}

// Global targets options. These options are independent of the per-probe
// targets which are defined by the "Targets" type above.
//
// Currently these options are used only for GCE targets to control things like
// how often to re-evaluate the targets and whether to check for lame ducks or
// not.
#GlobalTargetsOptions: {
	// RDS server address
	// Deprecated: This option is now deprecated, please use rds_server_options
	// instead.
	rdsServerAddress?: string @protobuf(3,string,name=rds_server_address,deprecated)

	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	rdsServerOptions?: proto.#ClientConf.#ServerOptions @protobuf(4,rds.ClientConf.ServerOptions,name=rds_server_options)

	// GCE targets options.
	globalGceTargetsOptions?: proto_5.#GlobalOptions @protobuf(1,gce.GlobalOptions,name=global_gce_targets_options)

	// Lame duck options. If provided, targets module checks for the lame duck
	// targets and removes them from the targets list.
	lameDuckOptions?: proto_8.#Options @protobuf(2,lameduck.Options,name=lame_duck_options)
}
