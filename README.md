[![Docker Pulls](https://img.shields.io/docker/pulls/cloudprober/cloudprober.svg)](https://hub.docker.com/v2/repositories/cloudprober/cloudprober/)
[![Go Build and Test](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml/badge.svg)](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)

Homepage and documentation: [cloudprober.org](https://cloudprober.org)

_NOTE: Cloudprober's active development moved from
~~[google/cloudprober](https://github.com/google/cloudprober)~~ to
[cloudprober/cloudprober](https://github.com/cloudprober/cloudprober) in
Nov, 2021. We lost a bunch of Github stars (1400) in the process. See
[story of cloudprober](https://medium.com/@manugarg/story-of-cloudprober-5ac1dbc0066c)
to learn more about the history of Cloudprober._


Cloudprober is a monitoring software that makes it super-easy to monitor
availability and performance of various components of your system. Cloudprober
employs the "active" monitoring model. It runs probes against (or on) your
components to verify that they are working as expected. For example, it can run
a probe to verify that your frontends can reach your backends. Similarly it can
run a probe to verify that your in-Cloud VMs can actually reach your on-premise
systems. This kind of monitoring makes it possible to monitor your systems'
interfaces regardless of the implementation and helps you quickly pin down
what's broken in your system (see
[why probers](https://medium.com/@manugarg/why-you-need-probers-f38400f5830e)).

<img width="460" src="https://cloudprober.org/homepage.png"/>

## Features

- Out of the box, config based, integration with many popular monitoring
  systems:

  - [Prometheus/Grafana](https://prometheus.io)
  - [DataDog](https://www.datadoghq.com/)
  - [PostgreSQL](https://www.postgresql.org/)
  - [AWS CloudWatch](https://aws.amazon.com/cloudwatch/)
  - [StackDriver / Google Cloud Monitoring](https://cloud.google.com/stackdriver/)

- Multiple options for checks:

  - _Efficient, highly scalable_, built-in probes:
    [HTTP](https://github.com/cloudprober/cloudprober/blob/master/probes/http/proto/config.proto),
    [PING](https://github.com/cloudprober/cloudprober/blob/master/probes/ping/proto/config.proto),
    [TCP](https://github.com/cloudprober/cloudprober/blob/master/probes/tcp/proto/config.proto),
    [DNS](https://github.com/cloudprober/cloudprober/blob/master/probes/dns/proto/config.proto),
    [gRPC](https://github.com/cloudprober/cloudprober/blob/master/probes/grpc/proto/config.proto),
    [UDP](https://github.com/cloudprober/cloudprober/blob/master/probes/udp/proto/config.proto).
  - Run custom checks through the
    _"[external](https://cloudprober.org/how-to/external-probe/)"_ probe type.

- Automated targets discovery to make Cloud deployments as painless as possible:

  - _[Kubernetes](https://cloudprober.org/how-to/run-on-kubernetes/#kubernetes-targets)_
    resources.
  - _[GCP](https://github.com/cloudprober/cloudprober/blob/master/internal/rds/gcp/proto/config.proto)_
    instances, forwarding rules, and pub/sub messages.
  - _[File](https://github.com/cloudprober/cloudprober/blob/master/internal/rds/file/proto/config.proto#L34)_
    based targets.

- Deployment friendly:

  - Written entirely in Go, and compiles into a static binary.
  - Deploy as a standalone binary, or through docker containers.
  - Continuous, automated target discovery, to ensure that most infrastructure
    changes don't require re-deployment.

- Low footprint. Cloudprober takes advantage of the Go's concurrency paradigms,
  and makes most of the available processing power.
- Configurable metrics:

  - Configurable metrics labels, based on the resource labels.
  - Latency histograms for percentile calculations.

- Extensible architecture. Cloudprober can be easily extended along most of the
  dimensions. Adding support for other Cloud targets, monitoring systems and
  even a new probe type, is straight-forward and fairly easy.

## Getting Started

Visit [Getting Started](http://cloudprober.org/getting-started) page to get
started with Cloudprober.

## Feedback

We'd love to hear your feedback. If you're using Cloudprober, would you please
mind sharing how you use it by adding a comment
[here](https://github.com/cloudprober/cloudprober/discussions/121). It will be a
great help in planning Cloudprober's future progression.

Join
[Cloudprober Slack](https://join.slack.com/t/cloudprober/shared_invite/enQtNjA1OTkyOTk3ODc3LWQzZDM2ZWUyNTI0M2E4NmM4NTIyMjM5M2E0MDdjMmU1NGQ3NWNiMjU4NTViMWMyMjg0M2QwMDhkZGZjZmFlNGE)
or [Github discussions](https://github.com/cloudprober/cloudprober/discussions)
for questions and discussion about Cloudprober.
