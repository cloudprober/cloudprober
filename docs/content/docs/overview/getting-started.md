---
menu:
  docs:
    parent: get-started
    weight: 2
title: Getting Started
---

## Installation

If you've Go installed, you can install cloudprober from source using the
following command:

```bash
go install github.com/cloudprober/cloudprober/cmd/cloudprober@latest
```

##### Other Installation Methods:

| Method             | Instructions                                                                                                                            | Platform              |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| Brew               | `brew install cloudprober`                                                                                                              | MacOS, Linux          |
| Docker Image       | `docker run ghcr.io/cloudprober/cloudprober` ([other docker versions](https://github.com/cloudprober/cloudprober/wiki/Docker-versions)) | Docker                |
| Helm chart         | [See instructions](https://github.com/cloudprober/helm-charts)                                                                            | Kubernetes            |
| Pre-built binaries | Download from the [releases page](http://github.com/cloudprober/cloudprober/releases).                                                  | MacOS, Linux, Windows |

See
[this page](https://github.com/cloudprober/cloudprober/wiki/Unreleased-Binaries)
for how to access unreleased binaries.

## Configuration

Without any config, cloudprober will run only the "sysvars" module (no probes)
and write metrics to stdout in cloudprober's line protocol format (to be
documented). It will also start a [Prometheus](http://prometheus.io) exporter
at: http://localhost:9313 (you can change the default port through the
environment variable _CLOUDPROBER_PORT_ and the default listening address
through the environment variable _CLOUDPROBER_HOST_).

Since sysvars variables are not very interesting themselves, lets add a simple
config that probes Google's homepage:

```bash
# Write config to a file in /tmp
cat > /tmp/cloudprober.cfg <<EOF
probe {
  name: "google_homepage"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }
  interval_msec: 5000  # 5s
  timeout_msec: 1000   # 1s
}
EOF
```

This config adds an HTTP probe that accesses the homepage of the target
"www.google.com" every 5s with a timeout of 1s. For more details on configs,
see the [config guide](/config/guide) and [reference](/config/latest/overview).

Assuming that you saved this file at `/tmp/cloudprober.cfg` (following the
command above), you can have cloudprober use this config file using the
following command line:

```bash
./cloudprober --config_file /tmp/cloudprober.cfg
```

You can have the standard docker image use this config using the following
command:

```bash
docker run -v /tmp/cloudprober.cfg:/etc/cloudprober.cfg \
    ghcr.io/cloudprober/cloudprober
```

Note: While running on GCE, cloudprober config can also be provided through a
custom metadata attribute: **cloudprober_config**.

## Verification

One quick way to verify that cloudprober got the correct config is to access the
URL `http://localhost:9313/config` (through cURL or in browser). It returns the
config that cloudprober is using. You can also look at its current status at the
URL (_replace localhost by the actual hostname if not running locally_):
`http://localhost:9313/status`.

You should be able to see the generated metrics at
`http://localhost:9313/metrics` (prometheus format) and the stdout (cloudprober
format):

```bash
cloudprober 15.. 1500590520 labels=ptype=http,probe=google-http,dst=.. total=17 success=17 latency=180835
cloudprober 15.. 1500590530 labels=ptype=sysvars,probe=sysvars hostname="manugarg-ws" uptime=100
cloudprober 15.. 1500590530 labels=ptype=http,probe=google-http,dst=.. total=19 success=19 latency=211644
```

This information is good for debugging monitoring issues, but to really make
sense of this data, you'll need to feed this data to another monitoring system
like _Prometheus_ or _StackDriver_ (see [Surfacers](/surfacers/overview) for
more details). Lets set up a Prometheus and Grafana stack to make pretty graphs
for us.

## Running Prometheus

Download prometheus binary from its
[release page](https://prometheus.io/download/). You can use a config like the
following to scrape a cloudprober instance running on the same host.

```bash
# Write config to a file in /tmp
cat > /tmp/prometheus.yml <<EOF
scrape_configs:
  - job_name: 'cloudprober'
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9313']
EOF

# Start prometheus:
./prometheus --config.file=/tmp/prometheus.yml
```

Prometheus provides a web interface at http://localhost:9090. You can explore
probe metrics and build useful graphs through this interface. All probes in
cloudprober export at least 3 counters:

- _total_: Total number of probes.
- _success_: Number of successful probes. Difference between _total_ and
  _success_ indicates failures.
- _latency_: Total (cumulative) probe latency.

Using these counters, probe failure ratio and average latency can be calculated
as:

```bash
failure_ratio = (rate(total) - rate(success)) / rate(total)
avg_latency   = rate(latency) / rate(success)
```

Assuming that prometheus is running at `localhost:9090`, graphs depicting
failure ratio and latency over time can be accessed in prometheus at:
[this url ](<"http://localhost:9090/graph?g0.range_input=1h&g0.expr=(rate(total%5B1m%5D)+-+rate(success%5B1m%5D))+%2F+rate(total%5B1m%5D)&g0.tab=0&g1.range_input=1h&g1.expr=rate(latency%5B1m%5D)+%2F+rate(success%5B1m%5D)+%2F+1000&g1.tab=0">).
Even though prometheus provides a graphing interface, Grafana provides much
richer interface and has excellent support for prometheus.

## Grafana

[Grafana](https://grafana.com) is a popular tool for building monitoring
dashboards. Grafana has native support for prometheus and thanks to the
excellent support for prometheus in Cloudprober itself, it's a breeze to build
Grafana dashboards from Cloudprober's probe results.

To get started with Grafana, follow the Grafana-Prometheus
[integration guide](https://prometheus.io/docs/visualization/grafana/).

## What's Next?

Explore Cloudprober's capabilities through the [configuration guide](
  /docs/config/guide) and [how to](/docs/how-to/list) pages. Checkout the
[example configs](
  https://github.com/cloudprober/cloudprober/tree/main/examples#cloudprober-examples)
in the Cloudprober GitHub repository. Finally, since not every feature is
documented, be sure to check out the [reference](
  /docs/config/latest/overview) documentation for complete details.