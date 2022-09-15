package proto

#SurfacerConf: {
	connectionString?:  string @protobuf(1,string,name=connection_string)
	metricsTableName?:  string @protobuf(2,string,name=metrics_table_name)
	metricsBufferSize?: int64  @protobuf(3,int64,name=metrics_buffer_size,"default=10000")
	labelToColumn?: [...#LabelToColumn] @protobuf(4,LabelToColumn,name=label_to_column)
}

#LabelToColumn: {
	label?:  string @protobuf(1,string)
	column?: string @protobuf(2,string)
}
