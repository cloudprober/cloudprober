---
menu:
  docs:
    parent: "faq"
    weight: 10
title: "General"
date: 2026-03-15
---

## What is Cloudprober?

Cloudprober is an open-source active monitoring tool that runs probes against
your infrastructure and applications, exports metrics, and optionally sends
alerts. It's written in Go and designed to be simple, efficient, and
cloud-native.

## How is Cloudprober different from Blackbox Exporter?

While both tools perform black-box probing, Cloudprober is a standalone
monitoring system with built-in support for dynamic target discovery,
multi-protocol probing, integrated alerting, and a status UI. Blackbox
Exporter relies on Prometheus to schedule probes, whereas Cloudprober manages
its own probe scheduling internally, making it more efficient at scale.

For a more detailed comparison, check out this [Medium blog post](
  https://medium.com/manugarg/prometheus-blackbox-exporter-vs-cloudprober-08a1d3beeda2) by Manu Garg.

## What protocols does Cloudprober support?

Cloudprober natively supports HTTP/HTTPS, DNS, PING (ICMP), TCP, and UDP
probes. For anything beyond these, you can use the EXTERNAL probe type to run
custom scripts or binaries as probes.

## Does Cloudprober support dynamic targets?

Yes. Cloudprober can automatically discover targets from Kubernetes, GCE,
file-based lists, and more. Targets are re-resolved periodically, so your probes
stay current as infrastructure changes.

## What metrics does Cloudprober export?

Every probe exports at least three metrics: `total` (probe attempts),
`success` (successful probes), and `latency` (cumulative latency in
microseconds). Probe-specific metrics (e.g., HTTP status codes, DNS
resolution details) are also available.

## How do I run Cloudprober?

The simplest way to get started:

```shell
# Using Go
go install github.com/cloudprober/cloudprober/cmd/cloudprober@latest
cloudprober --config_file cloudprober.cfg

# Using Docker
docker run --net host -v /path/to/cloudprober.cfg:/etc/cloudprober.cfg \
  ghcr.io/cloudprober/cloudprober
```

See the [Getting Started](/docs/overview/getting-started) guide for detailed
instructions.

## How do I change the default port?

Set the `CLOUDPROBER_PORT` environment variable:

```shell
CLOUDPROBER_PORT=8080 cloudprober --config_file cloudprober.cfg
```

## What configuration formats are supported?

Cloudprober supports textproto (`.cfg`), YAML (`.yaml`/`.yml`), JSON
(`.json`), and Jsonnet (`.jsonnet`) configuration formats. See the
[Configuration Guide](/docs/config/guide) for details on each format and
examples.

## Can I run Cloudprober on Kubernetes?

Yes. Cloudprober works well on Kubernetes and can be deployed using Helm
charts. It also supports Kubernetes-native target discovery. See the
[Running on Kubernetes](/docs/how-to/run-on-kubernetes) guide for more
details.
