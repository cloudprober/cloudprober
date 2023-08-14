package proto

// Dist defines a Distribution data type.
#Dist: {
	{} | {
		// Comma-separated list of lower bounds, where each lower bound is a float
		// value. Example: 0.5,1,2,4,8.
		explicitBuckets: string @protobuf(1,string,name=explicit_buckets)
	} | {
		// Exponentially growing buckets
		exponentialBuckets: #ExponentialBuckets @protobuf(2,ExponentialBuckets,name=exponential_buckets)
	}
}

// ExponentialBucket defines a set of num_buckets+2 buckets:
//   bucket[0] covers (−Inf, 0)
//   bucket[1] covers [0, scale_factor)
//   bucket[2] covers [scale_factor, scale_factor*base)
//   ...
//   bucket[i] covers [scale_factor*base^(i−2), scale_factor*base^(i−1))
//   ...
//   bucket[num_buckets+1] covers [scale_factor*base^(num_buckets−1), +Inf)
// NB: Base must be at least 1.01.
#ExponentialBuckets: {
	scaleFactor?: float32 @protobuf(1,float,name=scale_factor) // default = 1.0
	base?:        float32 @protobuf(2,float)                   // default = 2
	numBuckets?:  uint32  @protobuf(3,uint32,name=num_buckets) //default = 20
}
