---
menu:
  docs:
    parent: get-started
    name: "What is a Probe"
    weight: 3
title: "What is a Probe"
---

Cloudprober runs probes, but what is a probe? A probe runs an operation, usually
against a set of targets (e.g., your API servers), and looks for an expected
outcome. Typically probes access your systems the same way as your customers,
hence verifying systems' availability and performance from consumers' point of
view. For example, an HTTP probe executes an HTTP request against a web server
to verify that the web server is available. Cloudprober probes run repeatedly at
a configured interval and export probe results as a set of metrics.

```
Example of an HTTP Probe checking the frontend and API availability.
 _____________                   _______________
|             |   HTTP Probe    |               |
| Cloudprober |  ------------>  |  Website/APIs |
|_____________|                 |_______________|
```

Here are some of the options used to configure a probe:

| Field           | Description                                                     |
| --------------- | --------------------------------------------------------------- |
| `type`          | Probe type, for example: HTTP, PING or UDP                      |
| `name`          | Probe name. Each probe should have a unique name.               |
| `interval_msec` | How often to run the probe (in milliseconds).                   |
| `timeout_msec`  | Probe timeout (in milliseconds).                                |
| `targets`       | Targets to run probe against.                                   |
| `validator`     | Probe validators, further explained [here](/how-to/validators). |
| `<type>_probe`  | Probe type specific configuration, e.g. http_probe              |

Please take a look at the
[ProbeDef protobuf](/docs/config/probes/#cloudprober_probes_ProbeDef) for
further details on various fields and options. All probe types export at least
the following metrics:

| Metric    | Description                                                                                                                                   |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `total`   | Total probes run so far.                                                                                                                      |
| `success` | Number of successful probes. Deficit between _total_ and _success_ indicates failures.                                                        |
| `latency` | Cumulative probe latency (by default in microseconds). You can get more insights into latency by using [distributions](/how-to/percentiles/). |

Note that by default all metrics are cumulative, i.e. we export sum of all the
values so far. Cumulative metrics have this nice property that you don't lose
historical information if you miss a metrics read cycle, but they also make
certain calculations slightly more complicated (see below). To provide a choice
to the user, Cloudprober provides an option to export metrics as gauge values.
See [modifying metrics](/docs/surfacers/overview/#modifying-metrics) for more
details.

Example: In prometheus, you'll do something like the following to compute
success ratio and average latency from cumulative metrics.

```
success_ratio_1m = increase(success[1m]) / increase(total[1m])
average_latency_1m = increase(latency[1m]) / increase(success[1m])
```

## Probe Types

Cloudprober has built-in support for the following probe types:

- [HTTP](#http)
- [External](#external)
- [Ping](#ping)
- [DNS](#dns)
- [UDP](#udp)
- [TCP](#tcp)

More probe types can be added through
[cloudprober extensions](/docs/how-to/extensions).

### HTTP

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/http) |
[`Config options`](/docs/config/probes/#cloudprober_probes_http_ProbeConf)

HTTP probe sends HTTP(s) requests to a target and verify that a response is
received. Apart from the core probe metrics (total, success, and latency), HTTP
probes also export a map of response code counts (`resp_code`). By default,
requests are marked as successful as long as they succeed, regardless of the
HTTP response code, but this behavior can be changed by using
[validators](/docs/how-to/validators). For example, you can add a validator to
require status code to in a certain range, or response body to match a regex,
etc
([validator example](https://github.com/cloudprober/cloudprober/blob/master/examples/validators/cloudprober_validator.cfg)).

- **SSL Certificate Expiry**: If the target serves an SSL Certificate,
  cloudprober will walk the certificate chain and export the earliest expiry
  time in seconds as a metric. The metric is named
  `ssl_earliest_cert_expiry_sec`, and will only be exported when the expiry time
  in seconds is a positive number.

### External

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/external)
| [`Config options`](/docs/config/probes/#cloudprober_probes_external_ProbeConf)

External probe type allows running arbitrary programs for probing. This is
useful for running complex checks through Cloudprober. External probes are
documented in much more detail here:
[external probe](/docs/how-to/external-probe).

### Ping

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/ping) |
[`Config options`](/docs/config/probes/#cloudprober_probes_ping_ProbeConf)

Ping probe type implements a fast native ICMP ping prober, that can probe
hundreds of targets in parallel. Probe results are reported as number of packets
sent (total), received (success) and round-trip time (latency). It supports
both, privileged and unprivileged (uses ICMP datagram socket) pings.

Note that ICMP datagram sockets are not enabled by default on most Linux
systems. You can enable them by running the following command:
`sudo sysctl -w net.ipv4.ping_group_range="0 5000"`

### DNS

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/dns) |
[`Config options`](/docs/config/probes/#cloudprober_probes_dns_ProbeConf)

As the name suggests, DNS probe sends a DNS request to the target. This is
useful to verify that your DNS server, typically a critical component of the
infrastructure e.g. kube-dns, is working as expected.

### UDP

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/udp) |
[`Config options`](/docs/config/probes/#cloudprober_probes_udp_ProbeConf)

UDP probe sends a UDP packet to the configured targets. UDP probe (and all other
probes that use ports) provides more coverage for the network elements on the
data path as most packet forwarding elements use 5-tuple hashing and using a new
source port for each probe ensures that we hit different network element each
time.

### TCP

[`Code`](http://github.com/cloudprober/cloudprober/tree/master/probes/tcp) |
[`Config options`](/docs/config/probes/#cloudprober_probes_tcp_ProbeConf)

TCP probe verifies that we can establish a TCP connection to the given target
and port.
