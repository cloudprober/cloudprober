---
menu:
  docs:
    parent: 'config'
    weight: 02
title: 'Common Probe Options'
date: 2026-03-17T00:00:00-07:00
---

Every probe in Cloudprober shares a set of common options that control behavior
like targeting, timing, latency tracking, validation, alerting, and more. This
page covers these options with practical examples. For probe-type specific
options (HTTP headers, DNS query type, etc.), see the individual probe
documentation.

## Targets

Targets define what a probe checks. The simplest form is a comma-separated list
of hostnames or IPs:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org,github.com"
  }
  http_probe {}
}
```

Cloudprober also supports dynamic target discovery from Kubernetes, GCE, and
other sources -- targets are automatically added and removed as your
infrastructure changes. See the [Targets](/docs/how-to/targets) page for
details.

## IP Version

You can control which IP version a probe uses with `ip_version`. This affects
target name resolution, source IP selection (when using `source_interface`), and
packet crafting for PING probes.

```proto
probe {
  name: "web_v6"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  ip_version: IPV6
  http_probe {}
}
```

When `ip_version` is not set, the behavior depends on the probe type:

- **Packet-level probes (PING, UDP):** Default to **IPv4**, since they need to
  know the IP version upfront to craft the right packets.
- **Other probes (HTTP, TCP, DNS, gRPC, etc.):** Use whatever the system
  resolver returns -- typically both IPv4 and IPv6 addresses are available and
  the OS picks one.

If you set both `ip_version` and `source_ip`, they must be consistent -- e.g.,
an IPv6 source IP with `ip_version: IPV4` will cause an error.

## Latency Configuration

All probes export a cumulative `latency` metric. You can customize the unit,
metric name, and optionally track latency as a distribution (histogram) for
percentile analysis.

### Latency Unit

The default unit is microseconds (`us`). Change it with `latency_unit`:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  latency_unit: "ms"
  http_probe {}
}
```

Valid values: `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

### Latency Distribution

To get percentiles (p50, p95, p99), configure `latency_distribution` to bucket
latency values into a histogram:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  latency_unit: "ms"
  latency_distribution {
    explicit_buckets: "0.1,0.5,1,2,5,10,20,50,100"
  }
  http_probe {}
}
```

You can also use exponential buckets for automatic bucket generation:

```proto
latency_distribution {
  exponential_buckets {
    scale_factor: 0.1
    base: 2
    num_buckets: 15
  }
}
```

See [Percentiles and Histograms](/docs/how-to/percentiles/) for how to
visualize distributions in Prometheus and other backends.

### Latency Metric Name

If you use distributions for some probes and cumulative latency for others, you
can rename the metric to differentiate them:

```proto
probe {
  name: "web_latency_dist"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  latency_distribution {
    explicit_buckets: "0.5,1,2,5,10,25,50,100"
  }
  latency_metric_name: "latency_dist"
  http_probe {}
}
```

## Validators

Validators run checks on probe responses. All configured validators must pass
for a probe to be marked as success. For example, to verify both the HTTP
status code and response body:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }

  validator {
    name: "status_ok"
    http_validator {
      success_status_codes: "200-299"
    }
  }

  validator {
    name: "body_check"
    regex: "cloudprober"
  }

  http_probe {}
}
```

Validation failures are exported as a separate `validation_failure` metric with
a `validator` label, making it easy to identify exactly which check failed.

See [Validators](/docs/how-to/validators) for all validator types (HTTP status
codes, headers, regex, JSON, data integrity).

## Alerts

You can configure alerts directly on a probe to get notified when failures
exceed a threshold:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }

  alert {
    name: "web_down"
    condition {
      failures: 3
      total: 5
    }
    notify {
      slack {
        webhook_url: "https://hooks.slack.com/..."
      }
    }
  }

  http_probe {}
}
```

Cloudprober supports Slack, PagerDuty, Opsgenie, email, and generic HTTP
webhooks for notifications. See [Alerting](/docs/how-to/alerting) for the full
alerting guide.

## Stats Export Interval

Probes typically run at a high frequency (e.g., every 2 seconds), but stats are
aggregated before being exported. The default export interval is:

- **Most probes:** `max(interval, 10s)`
- **UDP probes:** `max(2 * max(interval, timeout), 10s)`

You can override this with `stats_export_interval_msec`:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  stats_export_interval_msec: 30000  # Export every 30s
  http_probe {}
}
```

A higher export interval reduces metric volume at the cost of resolution. In
most cases, the default works well.

## Negative Test

Negative tests invert the success criteria -- a probe counts as _successful_ if
the target _does not_ respond. This is useful for verifying that firewalls,
security rules, or access controls are working:

```proto
probe {
  name: "firewall_check"
  type: TCP
  targets {
    host_names: "internal-service.example.com"
  }
  negative_test: true
  tcp_probe {
    port: 8080
  }
}
```

If `internal-service.example.com:8080` is unreachable (as expected), the probe
reports success. If the connection succeeds, it reports failure -- indicating
the firewall may not be blocking traffic correctly.

_Note: Negative test is currently supported by PING, TCP, and HTTP probes. This
feature is experimental and may change._

## Scheduling

### Startup Delay

By default, Cloudprober staggers probe starts automatically (up to 1 minute).
For probes with very long intervals, you can add an explicit startup delay:

```proto
probe {
  name: "hourly_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  interval: "3600s"
  startup_delay_msec: 120000  # Wait 2 minutes before first run
  http_probe {}
}
```

This is additive to the automatic staggering and is useful for manually spacing
out probes.

### Schedule

Schedules let you control _when_ a probe runs based on time of day and day of
week. This is useful for probes that only make sense during business hours, or
for disabling probes during planned maintenance windows.

To run a probe only during business hours (Monday--Friday, 8am--6pm Eastern):

```proto
probe {
  name: "business_hours_check"
  type: HTTP
  targets {
    host_names: "internal-app.example.com"
  }

  schedule {
    type: ENABLE
    start_weekday: MONDAY
    start_time: "08:00"
    end_weekday: FRIDAY
    end_time: "18:00"
    timezone: "America/New_York"
  }

  http_probe {}
}
```

To disable a probe during a weekly maintenance window:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }

  schedule {
    type: DISABLE
    start_weekday: TUESDAY
    start_time: "02:00"
    end_weekday: TUESDAY
    end_time: "04:00"
    timezone: "America/New_York"
  }

  http_probe {}
}
```

You can combine multiple schedules. If ENABLE and DISABLE schedules overlap,
DISABLE takes precedence.

## Additional Labels

You can attach extra labels to a probe's metrics. Labels can be static values
or derived dynamically from the target or runtime environment:

```proto
probe {
  name: "web_check"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }

  # Static label
  additional_label {
    key: "team"
    value: "platform"
  }

  # Label from target's labels
  additional_label {
    key: "env"
    value: "@target.label.env@"
  }

  http_probe {}
}
```

This produces metrics like:

```
total{probe="web_check",dst="cloudprober.org",team="platform",env="prod"} 10
```

See [Additional Labels](/docs/how-to/additional-labels) for more examples
including runtime environment labels and global labels via environment
variables.
