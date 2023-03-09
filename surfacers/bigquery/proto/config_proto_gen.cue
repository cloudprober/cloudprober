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

	// Any column name passes here won't be inserted in bigquery like
	// 'metric_name', 'metric_value' or 'metric_time'
	excludeMetricColumn?: [...string] @protobuf(6,string,name=exclude_metric_column)
	metricsBufferSize?: int64 @protobuf(7,int64,name=metrics_buffer_size,"default=100000")

	// This denotes the time interval after which data will be inserted in
	// bigquery. Default is 10 seconds. So after every 10 seconds all the em in
	// current will be inserted in bigquery in a default batch size of 1000
	batchTimerSec?:    int64 @protobuf(8,int64,name=batch_timer_sec,"default=10")
	metricsBatchSize?: int64 @protobuf(9,int64,name=metrics_batch_size,"default=1000")
}

#BQColumn: {
	label?:      string @protobuf(1,string)
	columnName?: string @protobuf(2,string,name=column_name)
	columnType?: string @protobuf(3,string,name=column_type)
}
