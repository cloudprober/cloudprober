[![Docker Pulls](https://img.shields.io/docker/pulls/cloudprober/cloudprober.svg)](https://hub.docker.com/v2/repositories/cloudprober/cloudprober/)
[![Go Build and Test](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml/badge.svg)](https://github.com/cloudprober/cloudprober/actions/workflows/go.yml)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=cloudprober_cloudprober&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=cloudprober_cloudprober)

[cloudprober.org](https://cloudprober.org) | [Start Monitoring Now](https://cloudprober.org/getting-started)

# Cloudprober: Reliable System Monitoring, Simplified!

Cloudprober supercharges your monitoring with active probes (see
[why probers](https://medium.com/@manugarg/why-you-need-probers-f38400f5830e)) to ensure your systems—homelabs, microservices, APIs, websites, or cloud-to-on-prem connections—run smoothly. Detect failures before your users do.

<img width="460" src="https://cloudprober.org/homepage.png"/>

## Trusted by Leading Organizations
<table>
  <tr>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/google.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/tesla.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/snowflake.svg"></td>
    <td width="120" align="center"><img height=40 src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/apple.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/doordash.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/uber.svg"></td>
  </tr>
  <tr>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/cloudflare.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/walmart.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/robinhood.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/okta.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/gs.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/jpm.svg"></td>
  </tr>
  <tr>
    <td width="120" align="center"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/hostinger.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/digitalocean.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/disneyplus.svg"></td>
    <td width="120" align="center"><img src="https://raw.githubusercontent.com/cloudprober/cloudprober/main/docs/static/trusted-by-logos/yahoo-japan.svg"></td>
    <td width="120" align="center"></td>
  </tr>
</table>

## Why Cloudprober?

* Versatile Probes: Built-in HTTP, PING, TCP, DNS, gRPC, and UDP probes, plus custom checks via external probes.

* [Auto-Discover Targets](https://cloudprober.org/docs/how-to/targets/): Effortlessly monitor Kubernetes, GCP, or file-based resources without constant redeployment.

* Seamless Integrations: Out-of-the-box integration with Prometheus, Grafana, DataDog, AWS CloudWatch, PostgreSQL, and Google Cloud Monitoring.

* [Easy Alerts](https://cloudprober.org/docs/how-to/alerting/): Stay informed via email, Slack, PagerDuty, OpsGenie, or any other HTTP based system.

* Lightweight & Scalable: Written in Go, compiles to a single binary, and runs efficiently as a standalone app or Docker container.

* Custom Metrics: Flexible latency histograms and configurable labels for precise insights.

* Extensible: Easily add new probe types, targets, or monitoring systems.


## Get Started

Jump in with our [Getting Started](https://cloudprober.org/docs/overview/getting-started/) guide and start monitoring your systems in minutes.

## Join the Community

Join our [Slack](https://join.slack.com/t/cloudprober/shared_invite/enQtNjA1OTkyOTk3ODc3LWQzZDM2ZWUyNTI0M2E4NmM4NTIyMjM5M2E0MDdjMmU1NGQ3NWNiMjU4NTViMWMyMjg0M2QwMDhkZGZjZmFlNGE), or discuss on [Github](https://github.com/cloudprober/cloudprober/discussions). Help shape Cloudprober's future by commenting
[here](https://github.com/cloudprober/cloudprober/discussions/121).


_NOTE: Cloudprober's active development moved from
~~[google/cloudprober](https://github.com/google/cloudprober)~~ to
[cloudprober/cloudprober](https://github.com/cloudprober/cloudprober) in
Nov, 2021. We lost a bunch of Github stars (1400) in the process. See
[story of cloudprober](https://medium.com/@manugarg/story-of-cloudprober-5ac1dbc0066c)
to learn more about the history of Cloudprober._