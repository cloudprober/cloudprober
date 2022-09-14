package proto

#SurfacerConf: {
	connectionString?:  string @protobuf(1,string,name=connection_string)
	metricsTableName?:  string @protobuf(2,string,name=metrics_table_name)
	metricsBufferSize?: int64  @protobuf(3,int64,name=metrics_buffer_size,"default=10000")
}
