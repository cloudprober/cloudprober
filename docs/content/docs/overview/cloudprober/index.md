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

Cloudprober supercharges your monitoring with active probes (a.k.a. synthetic
monitoring) to ensure your systems—homelabs, microservices, APIs, websites, or
cloud-to-on-prem connections—run as expected. See
[this post](https://medium.com/@manugarg/why-you-need-probers-f38400f5830e) for
why probers provide one of the most reliable monitoring signals.

<img width="460" src="https://cloudprober.org/homepage.png"/>

## Why Cloudprober?

* **Versatile Probes**: Built-in HTTP, PING, TCP, DNS, gRPC, and UDP probes, plus custom checks via external probes.

* **Auto-Discover Targets**: Effortlessly monitor Kubernetes, GCP, or file-based resources without constant redeployment.

* **Integrate with Existing Systems**: Out-of-the-box integration with Prometheus, Grafana, DataDog, AWS CloudWatch, PostgreSQL, and Google Cloud Monitoring.

* **Easy Alerts**: Stay informed via email, Slack, PagerDuty, OpsGenie, or any other HTTP based system.

* **Lightweight & Scalable**: Written in Go, compiles to a single binary, and runs efficiently as a standalone app or Docker container.

* **Custom Metrics**: Flexible latency histograms and configurable labels for precise insights.

* **Extensible**: Easily add new probe types, targets, or monitoring systems.

## Learn More

* If you're familiar with Prometheus Blackbox Exporter, see how Cloudprober stacks against it: [Prometheus Blackbox Exporter vs Cloudprober](https://medium.com/cloudprober/prometheus-blackbox-exporter-vs-cloudprober-08a1d3beeda2).
  
* If you're not very familiar with the blackbox/synthetic monitoring paradigm, take a look at [why you need probers](https://medium.com/cloudprober/why-you-need-probers-f38400f5830e).

* Cloudprober's [origin story](https://medium.com/cloudprober/story-of-cloudprober-5ac1dbc0066c).
  
* How [DoorDash](https://careersatdoordash.com/blog/infra-prober-active-infrastructure-monitor/) & [Hostinger](https://www.hostinger.com/blog/cloudprober-explained-the-way-we-use-it-at-hostinger) use Cloudprober.

## Get Started

Jump in with our [Getting Started](https://cloudprober.org/docs/overview/getting-started/) guide and start monitoring your systems in minutes.

## Join the Community

Join our [Slack](https://join.slack.com/t/cloudprober/shared_invite/enQtNjA1OTkyOTk3ODc3LWQzZDM2ZWUyNTI0M2E4NmM4NTIyMjM5M2E0MDdjMmU1NGQ3NWNiMjU4NTViMWMyMjg0M2QwMDhkZGZjZmFlNGE), or discuss on [Github](https://github.com/cloudprober/cloudprober/discussions). Help shape Cloudprober's future by commenting
[here](https://github.com/cloudprober/cloudprober/discussions/121).
