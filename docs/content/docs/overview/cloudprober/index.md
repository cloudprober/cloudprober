---
title: Overview
menu:
  docs:
    parent: "get-started"
    params:
      hide: true
---

[![Docker Pulls](https://img.shields.io/docker/pulls/cloudprober/cloudprober.svg)](https://hub.docker.com/v2/repositories/cloudprober/cloudprober/)
[![Go Build and Test](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml/badge.svg)](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)

Cloudprober provides a reliable and easy-to-use solution to monitor the
availability and performance of your systems. Employing an "active" monitoring
approach, Cloudprober executes probes on or against these systems to verify
their proper functioning.

For example, you could use Cloudprober to run a probe to verify that your users
can access your website and your APIs, your microservices can talk to each
other, your kubernetes clusters can schedule pods, your CI/CD pipelines are
functioning as expected, or VPN connectivity with your partners is working as
expected, and much more.

This kind of monitoring makes it possible to monitor your systems' interfaces
regardless of the implementation, and helps you quickly pin down what's broken
in your systems.

<br/>

![Cloudprober Use Case](http://cloudprober.org/diagrams/cloudprober_use_case.svg)

<br/>

## Features

- Out of the box, config based, integration with many popular monitoring
  systems:

  - [Prometheus/Grafana](https://prometheus.io)
  - [DataDog](https://www.datadoghq.com/)
  - [PostgreSQL](https://www.postgresql.org/)
  - [AWS CloudWatch](https://aws.amazon.com/cloudwatch/)
  - [StackDriver / Google Cloud Monitoring](https://cloud.google.com/stackdriver/)

* Multiple options for checks:

  - _Efficient, highly scalable_, built-in probes:
    [HTTP](/docs/config/probes/#cloudprober_probes_http_ProbeConf),
    [PING](/docs/config/probes/#cloudprober_probes_ping_ProbeConf),
    [TCP](/docs/config/probes/#cloudprober_probes_tcp_ProbeConf),
    [DNS](/docs/config/probes/#cloudprober_probes_dns_ProbeConf),
    [gRPC](/docs/config/probes/#cloudprober_probes_grpc_ProbeConf),
    [UDP](/docs/config/probes/#cloudprober_probes_udp_ProbeConf).
  - Run custom checks through the _"[external]({{< ref external-probe >}})"_
    probe type.

- Automated [targets]({{< ref "/docs/how-to/targets.md" >}}) discovery to make
  Cloud deployments as painless as possible:

  - _[Kubernetes]({{< ref run-on-kubernetes.md >}}#kubernetes-targets)_
    resources.
  - _GCP_ instances, forwarding rules, and pub/sub messages.
  - _[File]({{< ref "/docs/how-to/targets.md" >}}#file-based-targets)_ based
    targets.

* Deployment friendly:

  - Written entirely in Go, and compiles into a static binary.
  - Deploy as a standalone binary, or through docker containers.
  - Continuous, automated target discovery, to ensure that most infrastructure
    changes don't require re-deployment.

- Low footprint. Cloudprober takes advantage of the Go's concurrency paradigms,
  and makes most of the available processing power.

* Configurable metrics:

  - Configurable metrics labels, based on the resource labels.
  - Latency histograms for percentile calculations.

- Extensible architecture. Cloudprober can be easily extended along most of the
  dimensions. Adding support for other Cloud targets, monitoring systems and
  even a new probe type, is straight-forward and fairly easy.

## Getting Started

Visit [Getting Started]({{< ref "getting-started.md" >}}) page to get started
with Cloudprober.

## Feedback

We'd love to hear your feedback. If you're using Cloudprober, would you please
mind sharing how you use it by adding a comment
[here](https://github.com/cloudprober/cloudprober/discussions/121). It will be a
great help in planning Cloudprober's future progression.

Join
[Cloudprober Slack](https://join.slack.com/t/cloudprober/shared_invite/enQtNjA1OTkyOTk3ODc3LWQzZDM2ZWUyNTI0M2E4NmM4NTIyMjM5M2E0MDdjMmU1NGQ3NWNiMjU4NTViMWMyMjg0M2QwMDhkZGZjZmFlNGE)
or [Github discussions](https://github.com/cloudprober/cloudprober/discussions)
for questions and discussion about Cloudprober.
