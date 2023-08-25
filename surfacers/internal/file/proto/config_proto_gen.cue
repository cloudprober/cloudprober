package proto

#SurfacerConf: {
	// Where to write the results. If left unset, file surfacer writes to the
	// standard output.
	filePath?: string @protobuf(1,string,name=file_path)
	prefix?:   string @protobuf(2,string,#"default="cloudprober""#)

	// Compress data before writing to the file.
	compressionEnabled?: bool @protobuf(3,bool,name=compression_enabled,"default=false")
}
