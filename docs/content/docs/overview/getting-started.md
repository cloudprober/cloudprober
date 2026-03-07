---
menu:
  docs:
    parent: get-started
    weight: 2
title: Getting Started
---

## Installation

If you have Go installed, you can install cloudprober from source:

```bash
go install github.com/cloudprober/cloudprober/cmd/cloudprober@latest
```

##### Other Installation Methods:

| Method             | Instructions                                                                                                                            | Platform              |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| Brew               | `brew install cloudprober`                                                                                                              | MacOS, Linux          |
| Docker Image       | `docker run ghcr.io/cloudprober/cloudprober` ([other docker versions](https://github.com/cloudprober/cloudprober/wiki/Docker-versions)) | Docker                |
| Helm chart         | [See instructions](https://github.com/cloudprober/helm-charts)                                                                          | Kubernetes            |
| Pre-built binaries | Download from the [releases page](http://github.com/cloudprober/cloudprober/releases).                                                  | MacOS, Linux, Windows |

Verify installation:

```bash
cloudprober --version
```

## Your First Run

Run cloudprober with no arguments:

```bash
cloudprober
```

On **Linux**, cloudprober automatically adds a `sys_metrics` system probe that
exports memory, CPU load, uptime, disk, and network metrics -- no config needed.
Open `http://localhost:9313/metrics` to see output like:

```
system_load_1m 0.52
system_mem_free 4.12e+09
system_uptime_sec 123456
system_disk_usage_free{disk="/dev/sda1"} 5.24e+10
system_net_rx_bytes{intf="eth0"} 1.02e+08
```

On non-Linux platforms, system metrics aren't auto-added, but all configured
probes still work.

Cloudprober also starts a built-in web UI at
`http://localhost:9313/status` where you can see probe status at a glance.

## Add Your First Probe

Create a config file that probes cloudprober.org every 5 seconds:

**Textproto** (`cloudprober.cfg`):

```
echo > /tmp/cloudprober.cfg <<EOF
probe {
  name: "cloudprober_website"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  http_probe {
    protocol: HTTPS
  }
  interval: "5s"
  timeout: "1s"
}
EOF
```
(Note: you can write the same config in YAML format too.
 See [config guide](/docs/config/guide/) to learn more.)


Run with your config:

```bash
cloudprober --config_file /tmp/cloudprober.cfg
```

Or if using Docker:

```bash
docker run -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
    ghcr.io/cloudprober/cloudprober
```

## See Your Results

Cloudprober exports metrics in two ways by default:

- **Prometheus endpoint** at `http://localhost:9313/metrics`
- **Stdout** in cloudprober's line format

Prometheus-format output looks like:

```
# HELP total Total probes
total{probe="cloudprober_website",dst="cloudprober.org"} 120
success{probe="cloudprober_website",dst="cloudprober.org"} 120
latency{probe="cloudprober_website",dst="cloudprober.org"} 2489734
```

**Built-in web endpoints:**

| Endpoint   | Description                      |
| ---------- | -------------------------------- |
| `/status`  | Probe status dashboard           |
| `/config`  | Current configuration            |
| `/metrics` | Prometheus-format metrics        |
| `/alerts`  | Active alerts                    |

You can change the default port (`9313`) with the `CLOUDPROBER_PORT` environment
variable and the listening address with `CLOUDPROBER_HOST`.

## More Probe Examples

Cloudprober supports several probe types. Here are a few common patterns:

**DNS probe** -- verify a DNS resolver:

```
probe {
  name: "dns_google"
  type: DNS
  targets {
    host_names: "8.8.8.8"
  }
  dns_probe {
    resolved_domain: "cloudprober.org"
  }
  interval: "10s"
}
```

**HTTP probe with validator** -- check an API returns 2xx:

```protobuf
probe {
  name: "api_health"
  type: HTTP
  targets {
    host_names: "api.example.com"
  }
  http_probe {
    protocol: HTTPS
    relative_url: "/health"
  }
  validator {
    http_validator {
      success_status_codes: "200-299"
    }
  }
  interval: "10s"
}
```

**Ping probe** -- monitor network reachability:

```protobuf
probe {
  name: "ping_dns"
  type: PING
  targets {
    host_names: "8.8.8.8,1.1.1.1"
  }
  interval: "5s"
}
```

See the [probe reference](/docs/overview/probe) and
[config reference](/docs/config/latest/overview) for all probe types and
options.

## Add Alerts

Cloudprober can alert on probe failures and supports Slack, PagerDuty,
Opsgenie, email, and HTTP webhooks.

Extend the earlier probe with an alert that notifies Slack after 2 consecutive
failures:

```protobuf
probe {
  name: "cloudprober_website"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  http_probe {
    protocol: HTTPS
  }
  interval: "5s"
  timeout: "1s"
  alert {
    name: "website_down"
    condition {
      failures: 2
    }
    notify {
      slack {
        webhook_url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
      }
    }
  }
}
```

See [Alerting](/docs/how-to/alerting) for full details on alert conditions and
notification options.

## Export Metrics

By default, cloudprober exposes a Prometheus-compatible endpoint at
`:9313/metrics`, ready for any Prometheus server to scrape.

Other supported export backends: **OpenTelemetry**, **CloudWatch**,
**Stackdriver**, **PostgreSQL**, **Pub/Sub**, **Datadog**, **BigQuery**.

All probes export at least three counters -- `total`, `success`, and `latency`.
Useful PromQL formulas:

```
success_ratio = rate(success[5m]) / rate(total[5m])
avg_latency   = rate(latency[5m]) / rate(success[5m])
```

See [Surfacers](/docs/surfacers/overview) for setup details on each backend.

## What's Next?

**Learn more:**

- [Configuration Guide](/docs/config/guide) -- formats, templates, modular
  configs
- [What is a Probe](/docs/overview/probe) -- types, metrics, options
- [Config Reference](/docs/config/latest/overview) -- complete field reference

**How-to guides:**

- [Alerting](/docs/how-to/alerting)
- [Validators](/docs/how-to/validators)
- [Running on Kubernetes](/docs/how-to/run-on-kubernetes)
- [External Probe](/docs/how-to/external-probe)

**Explore:**

- [Example Configs on GitHub](https://github.com/cloudprober/cloudprober/tree/main/examples#cloudprober-examples)
- [Community Slack](https://cloudprober.slack.com)
