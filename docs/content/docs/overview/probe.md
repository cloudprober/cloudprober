---
menu:
  docs:
    parent: get-started
    name: "What is a Probe"
    weight: 3
title: "What is a Probe"
---

## Active Monitoring with Probes

Most monitoring is _passive_ -- you collect logs, scrape application metrics, or
wait for errors to show up in dashboards. This tells you what's happening
_inside_ your systems, but it doesn't tell you what your users are actually
experiencing. A service can report healthy metrics internally while being
unreachable from the outside due to a DNS misconfiguration, a firewall rule, or
an expired certificate.

**Probes take the opposite approach.** A probe tests your system from the
outside, the same way a real user or customer would. An HTTP probe makes a real
HTTP request to your website. A DNS probe sends a real DNS query to your
resolver. A ping probe sends ICMP packets to check network reachability. If the
probe fails, you know your users are affected -- even if internal metrics look
fine.

This is what Cloudprober does: it runs probes continuously, checks the results,
and exports metrics and alerts so you know when something is wrong -- often
before your users notice.

```
 _____________        probe         _______________
|             |  -------------->  |               |
| Cloudprober |                   |  Your Systems |
|_____________|  <--------------  |_______________|
                  success/fail
                    latency
```

## How a Probe Works

Every probe in Cloudprober follows the same lifecycle:

1. **Repeat at an interval** -- A probe runs on a schedule (e.g., every 5
   seconds). Each run is one probe cycle.
2. **Check each target** -- Each cycle, the probe runs against all its
   configured targets (e.g., a list of hostnames or IPs).
3. **Record the outcome** -- Each probe attempt produces a result: did it
   succeed, and how long did it take?
4. **Export metrics** -- Results are exported as metrics that you can scrape with
   Prometheus, send to CloudWatch, Datadog, or any other
   [supported backend](/docs/surfacers/overview).
5. **Alert on failures** -- Optionally, probes can trigger
   [alerts](/docs/how-to/alerting) when failures exceed a threshold.

All probe types export at least three metrics:

| Metric    | Description                                             |
| --------- | ------------------------------------------------------- |
| `total`   | Number of probe attempts so far.                        |
| `success` | Number of successful probes. `total - success` = failures. |
| `latency` | Cumulative latency (microseconds by default).           |

These three metrics are enough to compute success ratio and average latency for
any probe. Individual probe types may export additional metrics (e.g., HTTP
probes export response code counts). See
[distributions](/docs/how-to/percentiles/) for latency percentiles.

## Probe Types

Cloudprober has built-in support for the following probe types. Pick the one
that matches how your users reach your system:

### HTTP

**Use for:** Websites, REST APIs, health check endpoints, anything served over
HTTP/HTTPS.

HTTP probes make real HTTP requests and verify a response is received. You can
use [validators](/docs/how-to/validators) to check status codes, response
bodies, or headers. HTTP probes also track response code counts and SSL
certificate expiry.

### Browser

**Use for:** End-to-end testing of web applications -- login flows, checkout
processes, multi-step user journeys.

Browser probes run real browser interactions using Playwright: navigating pages,
clicking buttons, filling forms. Cloudprober adds scheduling, metrics, artifact
management (screenshots, traces), and alerting on top, turning your Playwright
tests into continuous production monitoring. See
[browser probe](/docs/how-to/browser-probe) for details.

### DNS

**Use for:** Verifying DNS resolvers (e.g., kube-dns, cloud DNS) are responding
correctly.

DNS probes send DNS queries to a target and verify a response is received. This
is important because DNS is typically a critical dependency -- if your DNS is
down, everything that depends on it is effectively down too.

### Ping

**Use for:** Network reachability, host availability, latency baselines.

Ping probes send ICMP packets and measure round-trip time. Cloudprober's
built-in ping prober is fast and can probe hundreds of targets in parallel.

### TCP

**Use for:** Verifying that a service is accepting connections on a specific
port.

TCP probes check that a TCP connection can be established to the target. This is
useful for databases, caches, or any service that listens on a TCP port but
doesn't speak HTTP.

### UDP

**Use for:** UDP services, or when you want to test network paths more
thoroughly.

UDP probes send packets to the target. Since network load balancers typically use
5-tuple hashing, each probe with a different source port tests a potentially
different network path -- giving you broader coverage of your network
infrastructure.

### External

**Use for:** Custom checks that don't fit the built-in probe types.

External probes let you run any program or script as a probe. This is useful
when you need to test complex workflows, check application-specific logic, or
integrate with existing monitoring scripts. See
[external probe](/docs/how-to/external-probe) for details.

Additional probe types can be added through
[extensions](/docs/how-to/extensions).

## What's Next?

- **Try it out:** [Getting Started](/docs/overview/getting-started) walks you
  through installing Cloudprober and running your first probe.
- **Configure probes:** The [config guide](/docs/config/guide) covers all
  configuration formats and options.
- **Full reference:** See the
  [ProbeDef reference](/docs/config/probes/#cloudprober_probes_ProbeDef) for
  every probe field and option.
