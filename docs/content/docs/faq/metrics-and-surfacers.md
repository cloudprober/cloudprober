---
menu:
  docs:
    parent: "faq"
    weight: 30
title: "Metrics and Surfacers"
date: 2026-03-15
---

## What is a surfacer?

A surfacer is Cloudprober's mechanism for exporting metrics to external
systems. Think of it as a metrics backend. Cloudprober supports many surfacers
including Prometheus, OpenTelemetry, CloudWatch, Stackdriver, Datadog,
PostgreSQL, BigQuery, and more.

## What is the default metrics export behavior?

If you don't configure any surfacers, Cloudprober automatically enables the
Prometheus surfacer (serving metrics at `/metrics` on port 9313) and a
file-based surfacer. The ProbeStatus surfacer (powering the web UI) is always
enabled.

## How do I integrate Cloudprober with Prometheus?

Cloudprober exposes a Prometheus-compatible `/metrics` endpoint on port 9313
by default. Simply add Cloudprober as a scrape target in your Prometheus
configuration:

```yaml
scrape_configs:
  - job_name: cloudprober
    static_configs:
      - targets: ["localhost:9313"]
```

## Can I export metrics to multiple backends simultaneously?

Yes. You can configure multiple surfacers in the same Cloudprober
configuration, and metrics will be exported to all of them.

## How do I calculate success ratio in PromQL?

```
rate(success[5m]) / rate(total[5m])
```

## How do I calculate average latency in PromQL?

```
rate(latency[5m]) / rate(success[5m])
```

## Can I get latency percentiles (p50, p95, p99)?

Yes. Cloudprober supports a `distribution` metric type that creates
histograms for probe latencies. You configure bucket boundaries in your probe
config, and the resulting histogram can be used to compute percentiles in your
monitoring system (e.g., using `histogram_quantile` in PromQL).

See the [Percentiles, Histograms, and Distributions](/docs/how-to/percentiles)
guide for detailed setup and examples.

## Can I get latency breakdown for HTTP probes?

Yes. HTTP probes support breaking down overall latency into individual stages.
You can enable this with the `latency_breakdown` field in your HTTP probe
config:

```shell
probe {
  name: "web_latency"
  type: HTTP
  targets {
    host_names: "cloudprober.org"
  }
  http_probe {
    latency_breakdown: ALL_STAGES
  }
}
```

This exports additional metrics for each stage of the HTTP request:

- `dns_latency` — DNS resolution time
- `connect_latency` — TCP connection time
- `tls_handshake_latency` — TLS handshake time
- `req_write_latency` — Time to write the request
- `first_byte_latency` — Time to first response byte

You can also select specific stages instead of `ALL_STAGES`:

```shell
http_probe {
  latency_breakdown: [DNS_LATENCY, TLS_HANDSHAKE_LATENCY]
}
```

See the [HTTP probe config reference](
/docs/config/latest/probes/#cloudprober_probes_http_ProbeConf) for all
options.
