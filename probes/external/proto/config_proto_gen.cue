package proto

import "github.com/cloudprober/cloudprober/metrics/payload/proto"

#ProbeConf: {
	// External probes support two mode: ONCE and SERVER. In ONCE mode, external
	// command is re-executed for each probe run, while in SERVER mode, command
	// is run in server mode, re-executed only if not running already.
	#Mode: {"ONCE", #enumValue: 0} |
		{"SERVER", #enumValue: 1}

	#Mode_value: {
		ONCE:   0
		SERVER: 1
	}
	mode?: #Mode @protobuf(1,Mode,"default=ONCE")

	// Command.  For ONCE probes, arguments are processed for the following field
	// substitutions:
	// @probe@    Name of the probe
	// @target@   Hostname of the target
	// @address@  IP address of the target
	//
	// For example, for target ig-us-central1-a, /tools/recreate_vm -vm @target@
	// will get converted to: /tools/recreate_vm -vm ig-us-central1-a
	command?: string @protobuf(2,string)

	// Command environment variables. These are passed on to the external probe
	// process as environment variables.
	envVar?: {
		[string]: string
	} @protobuf(6,map[string]string,env_var)

	// Options for the SERVER mode probe requests. These options are passed on to
	// the external probe server as part of the ProbeRequest. Values are
	// substituted similar to command arguments for the ONCE mode probes.
	#Option: {
		name?:  string @protobuf(1,string)
		value?: string @protobuf(2,string)
	}
	options?: [...#Option] @protobuf(3,Option)

	// Export output as metrics, where output is the output returned by the
	// external probe process, over stdout for ONCE probes, and through ProbeReply
	// for SERVER probes. Cloudprober expects variables to be in the following
	// format in the output:
	// var1 value1 (for example: total_errors 589)
	outputAsMetrics?:      bool                        @protobuf(4,bool,name=output_as_metrics,default)
	outputMetricsOptions?: proto.#OutputMetricsOptions @protobuf(5,metrics.payload.OutputMetricsOptions,name=output_metrics_options)

	// (Only applicable to ONCE mode). Disable streaming output metrics. By
	// default, external probe will export output metrics as they are available
	// on the stdout. If this option is set to true, output metrics will be
	// exported only after the probe has completed.
	// New in version 0.13.4. This was true by default in previous versions.
	disableStreamingOutputMetrics?: bool @protobuf(7,bool,name=disable_streaming_output_metrics,"default=false")
}
