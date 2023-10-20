package proto

import (
	"github.com/cloudprober/cloudprober/internal/validators/http/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/validators/integrity/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/validators/json/proto"
)

#Validator: {
	name?: string @protobuf(1,string)
	{} | {
		httpValidator: proto.#Validator @protobuf(2,http.Validator,name=http_validator)
	} | {
		// Data integrity validator
		integrityValidator: proto_1.#Validator @protobuf(3,integrity.Validator,name=integrity_validator)
	} | {
		// JSON validator
		jsonValidator: proto_5.#Validator @protobuf(5,json.Validator,name=json_validator)
	} | {
		// Regex validator
		regex: string @protobuf(4,string)
	}
}
