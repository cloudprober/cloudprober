package proto

#SurfacerConf: {
	// default 60s
	resolutionSec?: int32 @protobuf(1,int32,name=resolution_sec,"default=60")

	// Number of points in each timeseries. This field dictates how far back
	// can you go up to (resolution_sec * timeseries_size). Note that higher
	// this number, more memory you'll use.
	timeseriesSize?: int32 @protobuf(2,int32,name=timeseries_size,"default=4320")

	// Max targets per probe.
	maxTargetsPerProbe?: int32 @protobuf(3,int32,name=max_targets_per_probe,"default=20")

	// ProbeStatus URL
	// Note that older default URL /probestatus forwards to this URL to avoid
	// breaking older default setups.
	url?: string @protobuf(4,string,#"default="/status""#)

	// Page cache time
	cacheTimeSec?: int32 @protobuf(5,int32,name=cache_time_sec,"default=2")

	// Probestatus surfacer is enabled by default. To disable it, set this
	// option.
	disable?: bool @protobuf(6,bool)
}
