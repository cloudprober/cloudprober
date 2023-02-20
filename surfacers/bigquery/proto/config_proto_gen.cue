package proto

#SurfacerConf: {
	projectName?:       string @protobuf(1,string,name=project_name)
	bigqueryDataset?:   string @protobuf(2,string,name=bigquery_dataset)
	bigqueryTable?:     string @protobuf(3,string,name=bigquery_table)
	metricsBufferSize?: int64  @protobuf(4,int64,name=metrics_buffer_size,"default=100000")

	// It represents the bigquery table columns.
	//  bigquery_columns {
	//    name: "id",
	//    type: "string",
	//  }
	bigqueryColumns?: [...#BQColumn] @protobuf(5,BQColumn,name=bigquery_columns)

	// This denotes the time interval after which data will be inserted in
	// bigquery. Default is 10 seconds. So after every 10 seconds all the em in
	// current will be inserted in bigquery in a batch size of 1000
	batchInsertionIntervalSec?: int64 @protobuf(6,int64,name=batch_insertion_interval_sec,"default=10")

	// It represents bigquery client timeout in seconds. So, if bigquery insertion
	// is not completed within this time period then the request will fail and the
	// failed rows will be retried later.
	bigqueryTimeoutSec?: int64 @protobuf(7,int64,name=bigquery_timeout_sec,"default=30")
}

#BQColumn: {
	name?: string @protobuf(1,string)
	type?: string @protobuf(2,string)
}
