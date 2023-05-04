package proto

#SurfacerConf: {
	projectName?:     string @protobuf(1,string,name=project_name)
	bigqueryDataset?: string @protobuf(2,string,name=bigquery_dataset)
	bigqueryTable?:   string @protobuf(3,string,name=bigquery_table)

	// It represents the bigquery table columns.
	//  bigquery_columns {
	//    label: "id",
	//    column_name: "id",
	//    column_type: "string",
	//  }
	bigqueryColumns?: [...#BQColumn] @protobuf(4,BQColumn,name=bigquery_columns)

	// It represents bigquery client timeout in seconds. So, if bigquery insertion
	// is not completed within this time period then the request will fail and the
	// failed rows will be retried later.
	bigqueryTimeoutSec?: int64 @protobuf(5,int64,name=bigquery_timeout_sec,"default=30")
	metricsBufferSize?:  int64 @protobuf(6,int64,name=metrics_buffer_size,"default=100000")

	// This denotes the time interval after which data will be inserted in
	// bigquery. Default is 10 seconds. So after every 10 seconds all the em in
	// current will be inserted in bigquery in a default batch size of 1000
	batchTimerSec?:    int64 @protobuf(7,int64,name=batch_timer_sec,"default=10")
	metricsBatchSize?: int64 @protobuf(8,int64,name=metrics_batch_size,"default=1000")

	// Column name for metrics name, value and timestamp
	metricTimeColName?:  string @protobuf(9,string,name=metric_time_col_name,#"default="metric_time""#)
	metricNameColName?:  string @protobuf(10,string,name=metric_name_col_name,#"default="metric_name""#)
	metricValueColName?: string @protobuf(11,string,name=metric_value_col_name,#"default="metric_value""#)
}

#BQColumn: {
	label?:      string @protobuf(1,string)
	columnName?: string @protobuf(2,string,name=column_name)
	columnType?: string @protobuf(3,string,name=column_type)
}
