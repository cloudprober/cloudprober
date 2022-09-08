package proto

#ServerConf: {
	port?: int32 @protobuf(1,int32)

	#Type: {
		// Echos the incoming packet back.
		// Note that UDP echo server limits reads to 4098 bytes. For messages longer
		// than 4098 bytes it won't work as expected.
		"ECHO"
		#enumValue: 0
	} | {
		// Discard the incoming packet. Return nothing.
		"DISCARD"
		#enumValue: 1
	}

	#Type_value: {
		ECHO:    0
		DISCARD: 1
	}
	type?: #Type @protobuf(2,Type)
}
