package proto

// DNS query types from https://en.wikipedia.org/wiki/List_of_DNS_record_types
#QueryType: {"NONE", #enumValue: 0} |
	{"A", #enumValue: 1} |
	{"NS", #enumValue: 2} |
	{"CNAME", #enumValue: 5} |
	{"SOA", #enumValue: 6} |
	{"PTR", #enumValue: 12} |
	{"MX", #enumValue: 15} |
	{"TXT", #enumValue: 16} |
	{"RP", #enumValue: 17} |
	{"AFSDB", #enumValue: 18} |
	{"SIG", #enumValue: 24} |
	{"KEY", #enumValue: 25} |
	{"AAAA", #enumValue: 28} |
	{"LOC", #enumValue: 29} |
	{"SRV", #enumValue: 33} |
	{"NAPTR", #enumValue: 35} |
	{"KX", #enumValue: 36} |
	{"CERT", #enumValue: 37} |
	{"DNAME", #enumValue: 39} |
	{"APL", #enumValue: 42} |
	{"DS", #enumValue: 43} |
	{"SSHFP", #enumValue: 44} |
	{"IPSECKEY", #enumValue: 45} |
	{"RRSIG", #enumValue: 46} |
	{"NSEC", #enumValue: 47} |
	{"DNSKEY", #enumValue: 48} |
	{"DHCID", #enumValue: 49} |
	{"NSEC3", #enumValue: 50} |
	{"NSEC3PARAM", #enumValue: 51} |
	{"TLSA", #enumValue: 52} |
	{"HIP", #enumValue: 55} |
	{"CDS", #enumValue: 59} |
	{"CDNSKEY", #enumValue: 60} |
	{"OPENPGPKEY", #enumValue: 61} |
	{"TKEY", #enumValue: 249} |
	{"TSIG", #enumValue: 250} |
	{"URI", #enumValue: 256} |
	{"CAA", #enumValue: 257} |
	{"TA", #enumValue: 32768} |
	{"DLV", #enumValue: 32769}

#QueryType_value: {
	NONE:       0
	A:          1
	NS:         2
	CNAME:      5
	SOA:        6
	PTR:        12
	MX:         15
	TXT:        16
	RP:         17
	AFSDB:      18
	SIG:        24
	KEY:        25
	AAAA:       28
	LOC:        29
	SRV:        33
	NAPTR:      35
	KX:         36
	CERT:       37
	DNAME:      39
	APL:        42
	DS:         43
	SSHFP:      44
	IPSECKEY:   45
	RRSIG:      46
	NSEC:       47
	DNSKEY:     48
	DHCID:      49
	NSEC3:      50
	NSEC3PARAM: 51
	TLSA:       52
	HIP:        55
	CDS:        59
	CDNSKEY:    60
	OPENPGPKEY: 61
	TKEY:       249
	TSIG:       250
	URI:        256
	CAA:        257
	TA:         32768
	DLV:        32769
}
#DNSProto: {"UDP", #enumValue: 0} |
	{"TCP", #enumValue: 1} |
	{"TCP_TLS", #enumValue: 2}

#DNSProto_value: {
	UDP:     0
	TCP:     1
	TCP_TLS: 2
}

#ProbeConf: {
	// Domain to use when making DNS queries
	resolvedDomain?: string @protobuf(1,string,name=resolved_domain,#"default="www.google.com.""#)

	// DNS Query Type
	queryType?: #QueryType @protobuf(3,QueryType,name=query_type,"default=MX")

	// Minimum number of answers expected. Default behavior is to return success
	// if DNS response status is NOERROR.
	minAnswers?: uint32 @protobuf(4,uint32,name=min_answers,"default=0")

	// Whether to resolve the target (target is DNS server here) before making
	// the request. If set to false, we hand over the target directly to the DNS
	// client. Otherwise, we resolve the target first to an IP address.  By
	// default we resolve first if it's a discovered resource, e.g., a k8s
	// endpoint.
	resolveFirst?: bool @protobuf(5,bool,name=resolve_first)

	// Which DNS protocol is used for resolution.
	dnsProto?: #DNSProto @protobuf(97,DNSProto,name=dns_proto,"default=UDP")

	// Requests per probe.
	// Number of DNS requests per probe. Requests are executed concurrently and
	// each DNS request contributes to probe results. For example, if you run two
	// requests per probe, "total" counter will be incremented by 2.
	requestsPerProbe?: int32 @protobuf(98,int32,name=requests_per_probe,"default=1")

	// How long to wait between two requests to the same target. Only relevant
	// if requests_per_probe is also configured.
	//
	// This value should be less than (interval - timeout) / requests_per_probe.
	// This is to ensure that all requests are executed within one probe interval
	// and all of them get sufficient time. For example, if probe interval is 2s,
	// timeout is 1s, and requests_per_probe is 10,  requests_interval_msec
	// should be less than 10ms.
	requestsIntervalMsec?: int32 @protobuf(99,int32,name=requests_interval_msec,"default=0")
}
