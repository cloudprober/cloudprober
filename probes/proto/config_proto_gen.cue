package proto

import (
	"github.com/cloudprober/cloudprober/targets/proto"
	proto_1 "github.com/cloudprober/cloudprober/metrics/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/validators/proto"
	proto_A "github.com/cloudprober/cloudprober/internal/alerting/proto"
	proto_8 "github.com/cloudprober/cloudprober/probes/ping/proto"
	proto_E "github.com/cloudprober/cloudprober/probes/http/proto"
	proto_B "github.com/cloudprober/cloudprober/probes/dns/proto"
	proto_36 "github.com/cloudprober/cloudprober/probes/external/proto"
	proto_9 "github.com/cloudprober/cloudprober/probes/udp/proto"
	proto_3 "github.com/cloudprober/cloudprober/probes/udplistener/proto"
	proto_A2 "github.com/cloudprober/cloudprober/probes/grpc/proto"
	proto_F "github.com/cloudprober/cloudprober/probes/tcp/proto"
)

// Next tag: 101
#ProbeDef: {
	// Probe name. It should be unique across all probes.
	name?: string @protobuf(1,string)

	#Type: {"PING", #enumValue: 0} |
		{"HTTP", #enumValue: 1} |
		{"DNS", #enumValue: 2} |
		{"EXTERNAL", #enumValue: 3} |
		{"UDP", #enumValue: 4} |
		{"UDP_LISTENER", #enumValue: 5} |
		{"GRPC", #enumValue: 6} |
		{"TCP", #enumValue: 7} | {
			// One of the extension probe types. See "extensions" below for more
			// details.
			"EXTENSION"
			#enumValue: 98
		} | {
			// USER_DEFINED probe type is for a one off probe that you want to compile
			// into cloudprober, but you don't expect it to be reused. If you expect
			// it to be reused, you should consider adding it using the extensions
			// mechanism.
			"USER_DEFINED"
			#enumValue: 99
		}

	#Type_value: {
		PING:         0
		HTTP:         1
		DNS:          2
		EXTERNAL:     3
		UDP:          4
		UDP_LISTENER: 5
		GRPC:         6
		TCP:          7
		EXTENSION:    98
		USER_DEFINED: 99
	}
	type?: #Type @protobuf(2,Type)

	// Interval between two probe runs in milliseconds.
	// Only one of "interval" and "inteval_msec" should be defined.
	// Default interval is 2s.
	intervalMsec?: int32 @protobuf(4,int32,name=interval_msec)

	// Interval between two probe runs in string format, e.g. 10s.
	// Only one of "interval" and "inteval_msec" should be defined.
	// Default interval is 2s.
	interval?: string @protobuf(16,string)

	// Timeout for each probe in milliseconds
	// Only one of "timeout" and "timeout_msec" should be defined.
	// Default timeout is 1s.
	timeoutMsec?: int32 @protobuf(5,int32,name=timeout_msec)

	// Timeout for each probe in string format, e.g. 10s.
	// Only one of "timeout" and "timeout_msec" should be defined.
	// Default timeout is 1s.
	timeout?: string @protobuf(17,string)

	// Targets for the probe. Targets are required for all probes except
	// for external, user_defined, and extension probe types.
	targets?: proto.#TargetsDef @protobuf(6,targets.TargetsDef)

	// Latency distribution. If specified, latency is stored as a distribution.
	latencyDistribution?: proto_1.#Dist @protobuf(7,metrics.Dist,name=latency_distribution)

	// Latency unit. Any string that's parseable by time.ParseDuration.
	// Valid values: "ns", "us" (or "µs"), "ms", "s", "m", "h".
	latencyUnit?: string @protobuf(8,string,name=latency_unit,#"default="us""#)

	// Latency metric name. You may want to change the latency metric name, if:
	// you're using latency_distribution for some probes, and regular metric for
	// other probes, and you want to differentiate between the two.
	// For example:
	//   probe {
	//     name: "web1_latency"
	//     latency_distribution: {...}
	//     latency_metric_name: "latency_dist"
	//     ...
	//   }
	//   probe {
	//     name: "app1"
	//     ...
	//   }
	latencyMetricName?: string @protobuf(15,string,name=latency_metric_name,#"default="latency""#)

	// Validators for this probe. Validators are run on the data returned by the
	// probe. See https://cloudprober.org/docs/how-to/validators/ for more info.
	validator?: [...proto_5.#Validator] @protobuf(9,validators.Validator)
	// Set the source IP to send packets from, either by providing an IP address
	// directly, or a network interface.
	{} | {
		sourceIp: string @protobuf(10,string,name=source_ip)
	} | {
		sourceInterface: string @protobuf(11,string,name=source_interface)
	}

	// IP version to use for networking probes. If specified, this is used while
	// 1) resolving a target, 2) picking the correct IP for the source IP if
	// source_interface option is provided, and 3) to craft the packet correctly
	// for PING probes.
	//
	// If ip_version is not configured but source_ip is provided, we get
	// ip_version from it. If both are  confgiured, an error is returned if there
	// is a conflict between the two.
	//
	// If left unspecified and both addresses are available in resolve call or on
	// source interface, IPv4 is preferred.
	// Future work: provide an option to prefer IPv4 and IPv6 explicitly.
	#IPVersion: {"IP_VERSION_UNSPECIFIED", #enumValue: 0} |
		{"IPV4", #enumValue: 1} |
		{"IPV6", #enumValue: 2}

	#IPVersion_value: {
		IP_VERSION_UNSPECIFIED: 0
		IPV4:                   1
		IPV6:                   2
	}
	ipVersion?: #IPVersion @protobuf(12,IPVersion,name=ip_version)

	// How often to export stats. Probes usually run at a higher frequency (e.g.
	// every second); stats from individual probes are aggregated within
	// cloudprober until exported. In most cases, users don't need to change the
	// default.
	//
	// By default this field is set in the following way:
	// For all probes except UDP:
	//   stats_export_interval=max(interval, 10s)
	// For UDP:
	//   stats_export_interval=max(2*max(interval, timeout), 10s)
	statsExportIntervalMsec?: int32 @protobuf(13,int32,name=stats_export_interval_msec)

	// Additional labels to add to the probe results. Label's value can either be
	// static or can be derived from target's labels.
	//
	// Example:
	//   additional_label {
	//     key: "app"
	//     value: "@target.label.app@"
	//   }
	// (More detailed example at: examples/additional_label/cloudprober.cfg)
	additionalLabel?: [...#AdditionalLabel] @protobuf(14,AdditionalLabel,name=additional_label)

	// (Experimental) If set, test is inversed, i.e. we count it as success if
	// target doesn't respond. This is useful, for example, that your firewall is
	// working as expected.
	//
	// This is currently implemented only by PING and TCP probes.
	// Note: This field is currently experimental, and may change in future.
	negativeTest?: bool @protobuf(18,bool,name=negative_test)

	// Alerts configuration. If specified, cloudprober will generate alerts on
	// probe failures. You can specify multiple alerts.
	// Example:
	//  alert {
	//    name: "alert1"
	//    condition {...}
	//    notify {
	//      pagerduty { ...}
	//    }
	//  }
	//  alert {
	//    name: "alert2"
	//    notify { ... }
	//  }
	alert?: [...proto_A.#AlertConf] @protobuf(19,alerting.AlertConf)
	{} | {
		pingProbe: proto_8.#ProbeConf @protobuf(20,ping.ProbeConf,name=ping_probe)
	} | {
		httpProbe: proto_E.#ProbeConf @protobuf(21,http.ProbeConf,name=http_probe)
	} | {
		dnsProbe: proto_B.#ProbeConf @protobuf(22,dns.ProbeConf,name=dns_probe)
	} | {
		externalProbe: proto_36.#ProbeConf @protobuf(23,external.ProbeConf,name=external_probe)
	} | {
		udpProbe: proto_9.#ProbeConf @protobuf(24,udp.ProbeConf,name=udp_probe)
	} | {
		udpListenerProbe: proto_3.#ProbeConf @protobuf(25,udplistener.ProbeConf,name=udp_listener_probe)
	} | {
		grpcProbe: proto_A2.#ProbeConf @protobuf(26,grpc.ProbeConf,name=grpc_probe)
	} | {
		tcpProbe: proto_F.#ProbeConf @protobuf(27,tcp.ProbeConf,name=tcp_probe)
	} | {
		// This field's contents are passed on to the user defined probe,
		// registered for this probe's name through probes.RegisterUserDefined().
		userDefinedProbe: string @protobuf(99,string,name=user_defined_probe)
	}

	// Which machines this probe should run on. If defined, cloudprober will run
	// this probe only if machine's hostname matches this value. This is useful
	// for large deployments, where you may want to use the same prober config
	// everywhere but run this probe only on a subset of machines.
	runOn?: string @protobuf(3,string,name=run_on)

	// Schedule for the probe. You can use a schedule to specify when a probe
	// should or should not run. This is useful for running probes only during
	// business hours.
	//
	// You can specify multiple schedules. Probe will not run if any of the
	// "DISABLE" schedules are active. If both "ENABLE" and "DISABLE" schedules
	// overlap, "DISABLE" takes precedence.
	//
	// For example, to disable a probe during weekends and on Tuesday between 7pm
	// and 8pm, e.g. for rollouts:
	//   schdule {
	//     type: DISABLE
	//     start_weekday: FRIDAY
	//     start_time: "20:00"
	//     end_weekday: SUNDAY
	//     end_time: "17:00"
	//     timezone: "America/New_York"
	//   }
	//   schdule {
	//     type: DISABLE
	//     start_weekday: TUESDAY
	//     start_time: "19:00"
	//     end_weekday: TUESDAY
	//     end_time: "20:00"
	//     timezone: "America/New_York"
	//   }
	schedule?: [...#Schedule] @protobuf(101,Schedule)

	// Debug options. Currently only used to enable logging metrics.
	debugOptions?: #DebugOptions @protobuf(100,DebugOptions,name=debug_options)
}

#AdditionalLabel: {
	key?: string @protobuf(1,string)

	// Value can either be a static value or can be derived from target's labels.
	// To get value from target's labels, use target.labels.<target's label key>
	// as value.
	value?: string @protobuf(2,string)
}

#Schedule: {
	#Weekday: {"EVERYDAY", #enumValue: 0} |
		{"SUNDAY", #enumValue: 1} |
		{"MONDAY", #enumValue: 2} |
		{"TUESDAY", #enumValue: 3} |
		{"WEDNESDAY", #enumValue: 4} |
		{"THURSDAY", #enumValue: 5} |
		{"FRIDAY", #enumValue: 6} |
		{"SATURDAY", #enumValue: 7}

	#Weekday_value: {
		EVERYDAY:  0
		SUNDAY:    1
		MONDAY:    2
		TUESDAY:   3
		WEDNESDAY: 4
		THURSDAY:  5
		FRIDAY:    6
		SATURDAY:  7
	}

	#ScheduleType: {"ScheduleType_UNSPECIFIED", #enumValue: 0} |
		{"ENABLE", #enumValue: 1} |
		{"DISABLE", #enumValue: 2}

	#ScheduleType_value: {
		ScheduleType_UNSPECIFIED: 0
		ENABLE:                   1
		DISABLE:                  2
	}
	type?: #ScheduleType @protobuf(1,ScheduleType)

	// Period start weekday. If not specified, it defaults to EVERYDAY.
	startWeekday?: #Weekday @protobuf(2,Weekday,name=start_weekday,"default=EVERYDAY")

	// Start time in 24 hour HH:MM format.
	startTime?: string @protobuf(3,string,name=start_time,#"default="00:00""#)

	// Period end weekday. If not specified, it defaults to EVERYDAY.
	endWeekday?: #Weekday @protobuf(4,Weekday,name=end_weekday,"default=EVERYDAY")

	// End time in 24 hour HH:MM format.
	endTime?: string @protobuf(5,string,name=end_time,#"default="23:59""#)

	// Timezone in which the probe should run. If not specified, it defaults to
	// UTC. Example: "America/New_York"
	timezone?: string @protobuf(6,string,#"default="UTC""#)
}

#DebugOptions: {
	// Whether to log metrics or not.
	logMetrics?: bool @protobuf(1,bool,name=log_metrics)
}
