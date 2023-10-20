package proto

import "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"

#Pods: {
}

#Endpoints: {
}

#Services: {
}

#Ingresses: {
}

// Kubernetes provider config.
#ProviderConfig: {
	// Namespace to list resources for. If not specified, we default to all
	// namespaces.
	namespace?: string @protobuf(1,string)

	// Pods discovery options. This field should be declared for the pods
	// discovery to be enabled.
	pods?: #Pods @protobuf(2,Pods)

	// Endpoints discovery options. This field should be declared for the
	// endpoints discovery to be enabled.
	endpoints?: #Endpoints @protobuf(3,Endpoints)

	// Services discovery options. This field should be declared for the
	// services discovery to be enabled.
	services?: #Services @protobuf(4,Services)

	// Ingresses discovery options. This field should be declared for the
	// ingresses discovery to be enabled.
	// Note: Ingress support is experimental and may change in future.
	ingresses?: #Ingresses @protobuf(5,Ingresses)

	// Label selectors to filter resources. This is useful for large clusters.
	// label_selector: ["app=cloudprober", "env!=dev"]
	labelSelector?: [...string] @protobuf(20,string,name=label_selector)

	// Kubernetes API server address. If not specified, we assume in-cluster mode
	// and get it from the local environment variables.
	apiServerAddress?: string @protobuf(91,string,name=api_server_address)

	// TLS config to authenticate communication with the API server.
	tlsConfig?: proto.#TLSConfig @protobuf(93,tlsconfig.TLSConfig,name=tls_config)

	// How often resources should be evaluated/expanded.
	reEvalSec?: int32 @protobuf(99,int32,name=re_eval_sec,"default=60") // default 1 min
}
