package proto

#Validator: {
	// Validate the data integrity of the response using a pattern that is
	// repeated throughout the length of the response, with last len(response) %
	// len(pattern) bytes being zero bytes.
	//
	// For example if response length is 100 bytes and pattern length is 8 bytes,
	// first 96 bytes of the response should be pattern repeated 12 times, and
	// last 4 bytes should be set to zero byte ('\0')
	{} | {
		// Pattern string for pattern repetition based integrity checks.
		// For example, cloudprobercloudprobercloudprober...
		patternString: string @protobuf(1,string,name=pattern_string)
	} | {
		// Pattern is derived from the first few bytes of the payload. This is
		// useful when pattern is not known in advance, for example cloudprober's
		// ping probe repeates the timestamp (8 bytes) in the packet payload.
		// An error is returned if response is smaller than pattern_num_bytes.
		patternNumBytes: int32 @protobuf(2,int32,name=pattern_num_bytes)
	}
}
