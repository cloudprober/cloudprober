---
title: "Surfacers"
menu:
  docs:
    parent: "surfacers"
    params:
      hide: true
---

One of the biggest strengths of cloudprober is that it can export data to
multiple monitoring systems, even simultaneously, just based on simple
configuration. Cloudprober does that using a built-in mechanism, called
surfacers. Each surfacer type implements interface for a specific monitoring
system, for example,
[_cloudwatch_](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_cloudwatch_SurfacerConf)
surfacer publishes data to AWS Cloudwatch. You can configure multiple surfacers
at the same time. If you don't specify any surfacer,
[_prometheus_](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_prometheus_SurfacerConf)
and
[_file_](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_file_SurfacerConf)
surfacers are enabled automatically.

Cloudprober currently supports following surfacer types:

- Prometheus
  ([config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_prometheus_SurfacerConf))
- [Stackdriver (Google Cloud Monitoring)](../stackdriver)
- Google Pub/Sub
  ([config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_pubsub_SurfacerConf))
- Postgres
  ([config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_postgres_SurfacerConf))
- File
  ([config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_file_SurfacerConf))
- [Cloudwatch (AWS Cloud Monitoring)](../cloudwatch)

Overall
[surfacers config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_SurfacerDef).

It's easy to add more surfacers without having to understand the internals of
cloudprober. You only need to implement the
[Surfacer interface](https://github.com/cloudprober/cloudprober/blob/7bc30b62e42f3fe4e8a2fb8cd0e87ea18b73aeb8/surfacers/surfacers.go#L87).

## Configuration

Adding surfacers to cloudprober is as easy as adding "surfacer" config stanzas
to your config, like the following:

```shell
# Enable prometheus and stackdriver surfacers.

# Make probe metrics available at the URL :<cloudprober_port>/metrics, for
# scraping by prometheus.
surfacer {
  type: PROMETHEUS

  prometheus_surfacer {
    # Following option adds a prefix to exported metrics, for example,
    # "total" metric is exported as "cloudprober_total".
    metrics_prefix: "cloudprober_"
  }
}

# Stackdriver (Google Cloud Monitoring) surfacer_ No other configuration
# is necessary if running on GCP.
surfacer {
  type: STACKDRIVER
}
```

## Filtering Metrics

You can control which metrics are published to a surfacer using the filtering
mechanisms. For example, you may want to publish only specific metrics to AWS
Cloudwatch to save on the costs.

### Filtering by Label

To filter metrics by labels, use one of the following keys in the surfacer
configuration:

- `allow_metrics_with_label` (`allowMetricsWithLabel` in yaml)
- `ignore_metrics_with_label` (`ignoreMetricsWithLabel` in yaml)

_Note: `ignore_metrics_with_label` takes precedence over
`allow_metrics_with_label`._

For example, to ignore all `sysvar` metrics:

```
surfacer {
  type: PROMETHEUS

  ignore_metrics_with_label {
    key: "probe",
    value: "sysvars",
  }
}
```

Or to only allow metrics from http probes:

```
surfacer {
  type: PROMETHEUS

  allow_metrics_with_label {
    key: "ptype",
    value: "http",
  }
}
```

#### Filtering by Metric Name

For certain surfacers, cloudprober can filter metrics by name as well. Surfacers
that support this functionality are:

- Cloudwatch
- Prometheus
- Stackdriver

Within the surfacer configuration, use one of the following options:

- `allow_metrics_with_name ` (`allowMetricsWithName` in yaml)
- `ignore_metrics_with_name` (`ignoreMetricsWithName` in yaml)

_Note: `ignore_metrics_with_name` takes precedence over
`allow_metrics_with_name`._

To filter out all `validation_failure` metrics by name:

```
surfacer {
  type: PROMETHEUS

  ignore_metrics_with_name: "validation_failure"
}
```
