# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: github.com/cloudprober/cloudprober/probes/external/proto/server.proto
"""Generated protocol buffer code."""
from google.protobuf.internal import builder as _builder
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\nEgithub.com/cloudprober/cloudprober/probes/external/proto/server.proto\x12\x0b\x63loudprober\"\x90\x01\n\x0cProbeRequest\x12\x12\n\nrequest_id\x18\x01 \x02(\x05\x12\x12\n\ntime_limit\x18\x02 \x02(\x05\x12\x31\n\x07options\x18\x03 \x03(\x0b\x32 .cloudprober.ProbeRequest.Option\x1a%\n\x06Option\x12\x0c\n\x04name\x18\x01 \x02(\t\x12\r\n\x05value\x18\x02 \x02(\t\"H\n\nProbeReply\x12\x12\n\nrequest_id\x18\x01 \x02(\x05\x12\x15\n\rerror_message\x18\x02 \x01(\t\x12\x0f\n\x07payload\x18\x03 \x01(\tB:Z8github.com/cloudprober/cloudprober/probes/external/proto')

_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, globals())
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'github.com.cloudprober.cloudprober.probes.external.proto.server_pb2', globals())
if _descriptor._USE_C_DESCRIPTORS == False:

  DESCRIPTOR._options = None
  DESCRIPTOR._serialized_options = b'Z8github.com/cloudprober/cloudprober/probes/external/proto'
  _PROBEREQUEST._serialized_start=87
  _PROBEREQUEST._serialized_end=231
  _PROBEREQUEST_OPTION._serialized_start=194
  _PROBEREQUEST_OPTION._serialized_end=231
  _PROBEREPLY._serialized_start=233
  _PROBEREPLY._serialized_end=305
# @@protoc_insertion_point(module_scope)
