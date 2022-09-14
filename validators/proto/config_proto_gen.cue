package proto

import (
	"github.com/cloudprober/cloudprober/validators/http/proto"
	proto_1 "github.com/cloudprober/cloudprober/validators/integrity/proto"
)

#Validator: {
	name?: string @protobuf(1,string)
	{} | {
		httpValidator: proto.#Validator @protobuf(2,http.Validator,name=http_validator)
	} | {
		// Data integrity validator
		integrityValidator: proto_1.#Validator @protobuf(3,integrity.Validator,name=integrity_validator)
	} | {
		// Regex validator
		regex: string @protobuf(4,string)
	}
}
