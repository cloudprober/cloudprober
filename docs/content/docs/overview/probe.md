---
menu:
  docs:
    parent: get-started
    name: "What is a Probe"
    weight: 3
title: "What is a Probe"
---

Cloudprober's main task is to run probes. A probe executes something, usually
against a set of targets, to verify that the systems are working as expected
from consumers' point of view. For example, an HTTP probe executes an HTTP
request against a web server to verify that the web server is available.
Cloudprober probes run repeatedly at a configured interval and export probe
results as a set of metrics.

A probe is defined as a set of the following fields:

| Field           | Description                                                     |
| --------------- | --------------------------------------------------------------- |
| `type`          | Probe type, for example: HTTP, PING or UDP                      |
| `name`          | Probe name. Each probe should have a unique name.               |
| `interval_msec` | How often to run the probe (in milliseconds).                   |
| `timeout_msec`  | Probe timeout (in milliseconds).                                |
| `targets`       | Targets to run probe against.                                   |
| `validator`     | Probe validators, further explained [here](/how-to/validators). |
| `<type>_probe`  | Probe type specific configuration.                              |

Please take a look at the
[ProbeDef protobuf](/docs/config/probes/#cloudprober_probes_ProbeDef) for
further details on various fields and options. All probe types export following
metrics at a minimum:

| Metric    | Description                                                                                                                                                                                                                                                                                                                                   |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `total`   | Total number of probes.                                                                                                                                                                                                                                                                                                                       |
| `success` | Number of successful probes. Deficit between _total_ and _success_ indicates failures.                                                                                                                                                                                                                                                        |
| `latency` | Cumulative probe latency (by default in microseconds). Latency can also be configured to be a [distribution](/how-to/percentiles/) (histogram) metric through a config option (`latency_distribution`). By default it's just the sum of the latencies observed so far. Average latency can be computed using _rate(latency) / rate(success)_. |

## Probe Types

Cloudprober has built-in support for the following probe types:

- [Ping](#ping)
- [HTTP](#http)
- [UDP](#udp)
- [DNS](#dns)
- [External](#external)

More probe types can be added through cloudprober extensions (to be documented).

### Ping

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/ping) |
[`Config options`](/docs/config/probes/#cloudprober_probes_ping_ProbeConf)

Ping probe type implements a fast ping prober, that can probe hundreds of
targets in parallel. Probe results are reported as number of packets sent
(total), received (success) and round-trip time (latency). It supports raw
sockets (requires root access) as well as datagram sockets for ICMP (doesn't
require root access).

ICMP datagram sockets are not enabled by default on most Linux systems. You can
enable them by running the following command:
`sudo sysctl -w net.ipv4.ping_group_range="0 5000"`

### HTTP

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/http) |
[`Config options`](/docs/config/probes/#cloudprober_probes_http_ProbeConf)

HTTP probe is be used to send HTTP(s) requests to a target and verify that a
response is received. Apart from the core probe metrics (total, success, and
latency), HTTP probes also export a map of response code counts. Requests are
marked as failed if there is a timeout.

- **SSL Certificate Expiry**: If the target serves a SSL Certificate,
  cloudprober will walk the certificate chain and export the earliest expiry
  time in seconds as a metric. The metric is named
  `ssl_earliest_cert_expiry_sec`, and will only be exported when the expiry time
  in seconds is a positive number.

### UDP

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/udp) |
[`Config options`](/docs/config/probes/#cloudprober_probes_udp_ProbeConf)

UDP probe sends a UDP packet to the configured targets. UDP probe (and all other
probes that use ports) provides more coverage for the network elements on the
data path as most packet forwarding elements use 5-tuple hashing and using a new
source port for each probe ensures that we hit different network element each
time.

### DNS

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/dns) |
[`Config options`](/docs/config/probes/#cloudprober_probes_dns_ProbeConf)

DNS probe type is implemented in a similar way as other probes except for that
it sends DNS requests to the target.

### External

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/external)
| [`Config options`](/docs/config/probes/#cloudprober_probes_external_ProbeConf)

External probe type allows running arbitrary probes through cloudprober. For an
external probe, actual probe logic resides in an external program; cloudprober
only manages the execution of that program and provides a way to export that
data through the standard channel.

External probe can be configured in two modes:

- **ONCE**: In this mode, an external program is executed for each probe run.
  Exit status of the program determines the success or failure of the probe.
  External probe can optionally be configured to interpret external program's
  output as metrics. This is a simple model but it doesn't allow the external
  program to maintain state and multiple forks can be expensive depending on the
  frequency of the probes.

- **SERVER**: In this mode, external program is expected to run in server mode.
  Cloudprober automatically starts the external program if it's not running at
  the time of the probe execution. Cloudprober and external probe process
  communicate with each other over stdin/stdout using protobuf messages defined
  in
  [probes/external/proto/server.proto](https://github.com/cloudprober/cloudprober/blob/master/probes/external/proto/server.proto).
