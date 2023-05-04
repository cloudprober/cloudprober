---
menu:
  docs:
    parent: how-to
    name: "Targets"
    weight: 12
title: "Targets"
date: 2016-10-25T17:24:32-07:00
---

Cloudprober probes usually run against some targets[^1] to check those targets'
status, such as an HTTP probe to your APIs servers, or PING/TCP probes to a
third-party provider to verify network connectivity to them. Each probe can have
multiple targets. If a probe has multiple targets, Cloudprober runs concurrent
probes against each target. This page further explains how targets work in
Cloudprober.

{{< figure src=targets.svg width=350 >}}

## Dynamically Discovered Targets

One of the core features of Cloudprober is the automatic and continuous
discovery of targets. This feature is especially critical for the dynamic
environments that today's cloud based deployments make possible. For exmaple in
a kubernetes cluster the number of pods and their IPs can change on the fly,
either in response to replica count changes or node failures. Automated targets
discovery makes sure that we don't have to reconfigure Cloudprober in response
to such events.

{{< figure src=targets2.svg width=350 >}}

[^1]:
    There are some cases where there is no explicit target, for example, you may
    run a probe to measure your CI system's performance, or run a complex probe
    that touches many endpoints.

## Targets Configuration

Cloudprober provides multiple ways to configure targets for a probe.

### Static targets

Static targets are the easiest and most straight-forward to configure:

```shell
probe {
  ...
  targets {
    host_names: "www.google.com,www.yahoo.com,cloudprober:9313"
  }
  ..
}
```

In the above config, probe will run concurrently against 3 hosts:
_www.google.com_, _www.yahoo.com_, and _cloudprober:9313_ (yes, you can specify
ports here for port-aware probes).

### File based targets

You can define your targets in a file and refer to them in Cloudprober through
that file. This file can be modified independently, and whenever that happens
cloudprober will reload it automatically.

Example configuration:

```bash
targets {
  file_targets {
    file_path: "/var/run/cloudprober/vips.json"
  }
}
```

In the targets file, resources should be specified in a specific format. Here is
an example of targets in JSON format:

```json
{
  "resource": [
    {
      "name": "switch-xx-1",
      "ip": "10.1.1.1",
      "port": 8080,
      "labels": {
        "device_type": "switch",
        "cluster": "xx"
      }
    },
    {
      "name": "switch-xx-2",
      "ip": "10.1.1.2",
      "port": 8081,
      "labels": {
        "cluster": "xx"
      }
    }
  ]
}
```

<span class=small>(You can also define targets in the textproto format: <a
href="https://github.com/cloudprober/cloudprober/blob/master/rds/file/testdata/targets1.textpb">example</a>.
Full example with cloudprober.cfg:
<a href="https://github.com/cloudprober/cloudprober/blob/master/examples/file_based_targets">file_based_targets</a>)</span>

Even if you don't intend to use the auto-reload feature of the file targets,
they can still be quite useful over static targets as they allow you to specify
additional details for targets. For example, specifying target's IP address in
the example above lets you tackle the case where you want to specify target's
name, let's say for better identification or for HTTP requests to work, but
don't want to rely on DNS for resolving its IP address.

### K8s targets

K8s targets are explained at [Kubernetes
Targets]({{< ref run-on-kubernetes.md >}}#kubernetes-targets).

### GCP targets

Since Cloudprober started at GCP, it's no surprise that Cloudprober has great
support for GCP targets. Cloudprober supports the following GCP resources:

- GCE Instances
- Forwarding Rules (regional and global)
- Cloud pub/sub (list of hostnames over cloud pub/sub)

TODO: Add more details on GCP targets.

## Probe configuration through target fields

| Field                | Probe Type                                   | Configuration                                                                                                                                                                |
| -------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `port`               | Port aware probes (HTTP, DNS, TCP, UDP, etc) | If a target has an associated port, for example, a Kubernetes endpoint, it will automatically be used for probing unless a port has been explicitly configured in the probe. |
| `label:relative_url` | HTTP                                         | If an explicit relative URL is not set, HTTP probe will use `relative_url` label's value if set.                                                                             |
| `label:fqdn`         | HTTP                                         | HTTP probe will use target's `fqdn` label as the URL-host (host part of the URL) and Host header if available and if Host header has not been configured explicitly.         |

## Metrics

- Target name: All metrics generated by Cloudprober have a `dst` label which is
  set to the target name.
- Target labels: See [additional labels]({{< ref "additional-labels.md" >}}) for
  how resource labels can be used to set additional labels on the metrics.

## Scaling targets discovery and other features

If you run a lot of Cloudprober instances with targets discovery, you may end up
overwhelming the API servers, or running out of your API quota in case of Cloud
resources. To avoid that, Cloudprober allows centralizing the targets discovery
through the Resource Discovery Service (RDS) mechanism. See [Resource Discovery
Service]({{< ref rds >}}) for more details on that.

Other salient features of the cloudprober's targets discovery:

- Continuous discovery. We don't just discover targets in the beginning, but
  keep refreshing them at a regular interval.
- Protection against the upstream provider failures. If refreshing of the
  targets fails during one of the refresh cycles, we continue using the existing
  set of targets.
