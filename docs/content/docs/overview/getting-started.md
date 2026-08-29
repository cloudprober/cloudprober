---
menu:
  docs:
    parent: get-started
    weight: 2
title: Getting Started
---

## Installation

```bash
curl -fsSL https://cloudprober.org/install.sh | sh
```

This downloads the latest release for your platform, verifies its SHA-256
checksum, and installs the binary into `/usr/local/bin` (or `~/.local/bin` if
that isn't writable). Set `VERSION` or `INSTALL_DIR` to override either
default. Linux and macOS only.

If you'd rather not pipe a script into a shell -- and it's a good habit not to
-- read it first at
[cloudprober.org/install.sh](https://cloudprober.org/install.sh), or use one of
the methods below.

##### Other Installation Methods:

| Method             | Instructions                                                                                                                            | Platform              |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| Go                 | `go install github.com/cloudprober/cloudprober/cmd/cloudprober@latest`                                                                  | MacOS, Linux, Windows |
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

```text
system_load_1m{probe="sys_metrics",ptype="system"} 0.170 1756440000000
system_uptime_sec{probe="sys_metrics",ptype="system"} 258162.780 1756440000000
system_mem_available{probe="sys_metrics",ptype="system"} 56676487168.000 1756440000000
system_net_aggregated_rx_bytes{probe="sys_metrics",ptype="system"} 1899739498.000
system_disk_io_aggregated_read_bytes{probe="sys_metrics",ptype="system"} 49954401280.000
```

Disk and network stats are aggregated across all devices by default. To get a
separate series per mount point (`mount_point` label) or per interface (`iface`
label), configure a `SYSTEM` probe yourself with `export_individual_stats: true`.

Pass `--disable_sys_metrics` if you'd rather not have this probe added, or
configure a `SYSTEM` probe of your own -- cloudprober skips the automatic one
if your config already has a `SYSTEM` probe.

On non-Linux platforms, system metrics aren't auto-added, but all configured
probes still work.

Cloudprober also starts a built-in web UI at
`http://localhost:9313/status` where you can see probe status at a glance.

## Add Your First Probe

Create a config file that probes cloudprober.org every 5 seconds:

**Textproto** (`cloudprober.cfg`):

```bash
cat > cloudprober.cfg <<EOF
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
cloudprober --config_file cloudprober.cfg
```

If you don't pass `--config_file`, cloudprober reads `/etc/cloudprober.cfg`,
and starts with an empty config if that file doesn't exist either -- which is
what happened in [Your First Run](#your-first-run) above.

Or, with Docker. Note that the config is mounted at that same default path:

```bash
docker run -p 9313:9313 -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
    ghcr.io/cloudprober/cloudprober
```

## See Your Results

Cloudprober exports metrics in two ways by default:

- **Prometheus endpoint** at `http://localhost:9313/metrics`
- **Stdout** in cloudprober's line format

Prometheus-format output looks like:

```text
# TYPE total counter
total{ptype="http",probe="cloudprober_website",dst="cloudprober.org"} 120
# TYPE success counter
success{ptype="http",probe="cloudprober_website",dst="cloudprober.org"} 120
# TYPE latency counter
latency{ptype="http",probe="cloudprober_website",dst="cloudprober.org"} 639773.455
```

**Built-in web endpoints:**

| Endpoint          | Description                                       |
| ----------------- | ------------------------------------------------- |
| `/status`         | Probe status dashboard                            |
| `/logs`           | Recent log entries                                |
| `/metrics`        | Prometheus-format metrics                         |
| `/alerts`         | Active alerts                                     |
| `/config`         | Config exactly as you provided it                 |
| `/config-parsed`  | Config after templates and env vars are expanded  |
| `/config-running` | Running probes, surfacers, and servers            |
| `/links`          | Index of all of the above                         |

`/artifacts` also shows up when a probe produces artifacts, such as the
screenshots from a browser probe.

You can change the default port (`9313`) with the `CLOUDPROBER_PORT` environment
variable and the listening address with `CLOUDPROBER_HOST`.

## More Probe Examples

Cloudprober supports several probe types. Here are a few common patterns:

**DNS probe** -- verify a DNS resolver:

```protobuf
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

  ping_probe {
    # See the following for how this field works and permissions issues with
    # ping probes
    # https://cloudprober.org/goto/ping-permission-issue
    use_datagram_socket: false
  }
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
Useful PromQL queries:

```promql
# Success ratio, over a 5m window.
rate(success[5m]) / rate(total[5m])

# Average latency. Note that latency is in microseconds unless you change
# the probe's latency_unit.
rate(latency[5m]) / rate(success[5m])
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
- [Community Slack](/goto/slack-invite/)
