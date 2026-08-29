---
title: Overview
menu:
  docs:
    parent: "get-started"
    params:
      hide: true
---

Cloudprober supercharges your monitoring with active probes (a.k.a. synthetic
monitoring) to ensure your systems—homelabs, microservices, APIs, websites, or
cloud-to-on-prem connections—run as expected.

<img width="560" src="/homepage.png"
     alt="Cloudprober probes websites, internal services and partner
          connectivity, and feeds dashboards, SLOs and alerts"/>

Probes run continuously from wherever you choose to run them, measuring what
your users actually experience rather than what your servers report about
themselves. See [why you need probers](https://medium.com/cloudprober/why-you-need-probers-f38400f5830e)
for why that turns out to be one of the most reliable monitoring signals you
can have.

## Ready to try it?

[Getting Started](/docs/overview/getting-started/) has you probing a real
endpoint in a couple of minutes: install, one small config file, done.

## What's in the box

| | What you get |
| --- | --- |
| [Probes](/docs/overview/probe/) | HTTP, gRPC, Browser, Starlark script, DNS, PING, TCP, and UDP, plus [external](/docs/how-to/external-probe/) commands and Go [extensions](/docs/how-to/extensions/) when you need something else. |
| [Targets](/docs/how-to/targets/) | Discovered automatically from Kubernetes, GCP, or files, so you're not redeploying every time your fleet changes. |
| [Surfacers](/docs/surfacers/overview/) | Metrics out to Prometheus, Grafana, Datadog, CloudWatch, PostgreSQL, Google Cloud Monitoring, and more. |
| [Alerts](/docs/how-to/alerting/) | Email, Slack, PagerDuty, Opsgenie, or any HTTP endpoint. |
| [Config](/docs/config/guide/) | Textproto or YAML, with Go templates for when one probe block has to cover hundreds of endpoints. |

Written in Go, it compiles to a single binary and runs as happily on a
Raspberry Pi as it does across a global fleet.

## Learn more

* Coming from Prometheus Blackbox Exporter? See how the two compare:
  [Prometheus Blackbox Exporter vs Cloudprober](https://medium.com/cloudprober/prometheus-blackbox-exporter-vs-cloudprober-08a1d3beeda2).

* New to the blackbox/synthetic monitoring paradigm?
  [Why you need probers](https://medium.com/cloudprober/why-you-need-probers-f38400f5830e).

* Cloudprober's [origin story](https://medium.com/cloudprober/story-of-cloudprober-5ac1dbc0066c).

* How [DoorDash](https://careersatdoordash.com/blog/infra-prober-active-infrastructure-monitor/)
  and [Hostinger](https://www.hostinger.com/blog/cloudprober-explained-the-way-we-use-it-at-hostinger)
  use Cloudprober.

## Join the community

Join our [Slack](/goto/slack-invite/), or discuss on
[GitHub](https://github.com/cloudprober/cloudprober/discussions). And if you're
already running it, tell us about it in
[How do you use Cloudprober?](https://github.com/cloudprober/cloudprober/discussions/121)
