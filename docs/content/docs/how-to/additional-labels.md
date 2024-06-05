---
menu:
  docs:
    parent: "how-to"
    weight: 22
title: "Additional Labels"
date: 2022-10-01T17:24:32-07:00
---

You can add additional labels to Cloudprober metrics using two methods:

- Probe level config field: `additional_label`
- Through an environment variable: `CLOUDPROBER_ADDITIONAL_LABELS`.

## Probe Level Additional Labels

You can add additional labels to a specific probe's metrics using probe-level
config field: `additional_label`. An additional label's value can be static, or
it can be determined at the run-time: from the environment that the probe is
running in (e.g. GCE instance labels, or location), or target's labels.

Example config
[here](https://github.com/cloudprober/cloudprober/blob/master/examples/additional_label/cloudprober.cfg)
demonstrates adding various types of additional labels to probe metrics. For
this config (also listed below for quick rerefence):

- if ingress target has label "`fqdn:app.example.com`",
- and prober is running in the GCE zone `us-east1-c`,
- and prober's GCE instance has label `env:prod`.

Probe metrics will look like the following:

```
 total{probe="my_ingress",ptype="http",metrictype="prober",env="prod",src_zone="us-east1-c",host="app.example.com"}: 90
 success{probe="my_ingress",ptype="http",metrictype="prober",env="prod",src_zone="us-east1-c",host="app.example.com"}: 80
```

```bash
probe {
  name: "my_ingress"
  type: HTTP

  targets {
    k8s {
      ingresses: ""
      namespace: "default"
    }
  }

  # Static label
  additional_label {
    key: "metrictype"
    value: "prober"
  }

  # Label is configured at the run-time, based on the prober instance label (GCE).
  additional_label {
    key: "env"
    value: "{{.label_env}}"
  }

  # Label is configured at the run-time, based on the prober environment (GCE).
  additional_label {
    key: "src_zone"
    value: "{{.zone}}"
  }

  # Label is configured based on the target's labels.
  additional_label {
    key: "host"
    value: "@target.label.fqdn@"
  }

  http_probe {}
}
```

(Listing source:
[examples/additional_label/cloudprober.cfg](https://github.com/cloudprober/cloudprober/blob/master/examples/additional_label/cloudprober.cfg))

## Environment Based Additional Labels

You can also add labels to all metrics exported by cloudprober using an
environment variable: `CLOUDPROBER_ADDITIONAL_LABELS`. You can choose a
different environemnt variable or disable this behavior completely by modifying
the `additional_labels_env_var` config field in the
[surfacers config](/docs/config/surfacer/#cloudprober_surfacer_SurfacerDef).

Value of the environment variable should be a comma separated list of
`key=value` pairs. For example if CLOUDPROBER_ADDITIONAL_LABELS is set to
`app=ingester,env=prod`, cloudprober will add the following two labels to all
the metrics: `{env=prod, app=ingester}`.

This feature is particularly useful in a multi-single-tenant setup (where you
run one instance per tenant) to filter out metrics by tenant. For example, you
can add `tenant=team1` label to all cloudprober metrics by setting
`CLOUDPROBER_ADDITIONAL_LABELS=tenant=team1` in team1's pod's environment.

Note that existing probe labels have precedence over environment based labels.
If probe metrics already have a label (e.g. `dst`), and you try to add the same
label through this method, it will be silently ignored.

## Related

See [Exporting Metrics](/docs/how-to/additional-labels) to learn more about how
metrics are exported from Cloudprober.
