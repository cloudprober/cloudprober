---
menu:
  docs:
    parent: "faq"
    weight: 40
title: "Troubleshooting"
date: 2026-03-15
---

## PING probe fails with "socket: permission denied"

This is a common issue on Linux. By default, Cloudprober uses datagram
(unprivileged) ICMP sockets for ping probes, but most Linux distributions
don't allow unprivileged ICMP pings out of the box.

You have two options:

**Option A: Enable unprivileged pings (recommended)**

Allow your user's group to send ICMP pings:

```shell
sudo sysctl -w net.ipv4.ping_group_range="0 65535"
```

To make this persistent across reboots, add the following to
`/etc/sysctl.conf` or a file in `/etc/sysctl.d/`:

```
net.ipv4.ping_group_range = 0 65535
```

**Option B: Use raw sockets with CAP_NET_RAW**

If you prefer to use raw ICMP sockets, set `use_datagram_socket` to `false`
in your ping probe config and grant the binary the `CAP_NET_RAW` capability:

```proto
probe {
  name: "ping_dns"
  type: PING
  targets {
    host_names: "8.8.8.8,1.1.1.1"
  }
  ping_probe {
    use_datagram_socket: false
  }
}
```

```shell
sudo setcap cap_net_raw+ep ./cloudprober
```

Note: Setting `cap_net_raw` alone without `use_datagram_socket: false` will
**not** work, because the default datagram socket path doesn't use raw
sockets.

**Running in Docker**

You have two options in Docker as well:

*Option A (recommended):* Enable unprivileged pings by passing the sysctl to
the container:

```shell
docker run --sysctl net.ipv4.ping_group_range="0 65535" \
  -v /path/to/cloudprober.cfg:/etc/cloudprober.cfg \
  ghcr.io/cloudprober/cloudprober
```

Note: `--sysctl` requires the container to have its own network namespace
(the default). It won't work with `--net host`; in that case, set the sysctl
on the host instead.

*Option B:* Use raw sockets by adding the `NET_RAW` capability and setting
`use_datagram_socket: false` in your probe config:

```shell
docker run --cap-add NET_RAW \
  -v /path/to/cloudprober.cfg:/etc/cloudprober.cfg \
  ghcr.io/cloudprober/cloudprober
```

## Cloudprober is running but I don't see any metrics

Check the following:

1. **Verify probes are running:** Visit `http://localhost:9313/status` to see
   probe status.
2. **Check the metrics endpoint:** Visit `http://localhost:9313/metrics` to
   see raw Prometheus metrics.
3. **Check for errors in logs:** Cloudprober logs errors at startup if probes
   fail to initialize.
4. **Verify surfacer config:** Visit `http://localhost:9313/config` to see
   running config. If you configured surfacers explicitly, make sure the default
   Prometheus surfacer wasn't disabled. 

## "no suitable address found" or "no IPv4 address" errors with IPv6 targets

If you're monitoring IPv6 targets and see errors like:

```
Resolve Error: address ::1: no suitable address found
```

or:

```
Bad target: myhost. Err: no IPv4 address (IP: 2001:db8::1) for myhost
```

This happens because some probe types (like PING and UDP) need to know the IP
version to craft appropriate packets and default to IPv4 when `ip_version` is
not set. If you have packet-level probes that use IPv6 addresses, you need to
explicitly set `ip_version` in your probe configuration:

```proto
probe {
  name: "ping_v6"
  type: PING
  targets {
    host_names: "::1"
  }
  ip_version: IPV6
  ping_probe {
    ...
    use_datagram_socket: false
  }
}
```

Note that IPv4 and IPv6 targets cannot be mixed in a single probe. If you need
to monitor both, create separate probes for each IP version.

## Probes show high latency or timeouts

- Ensure the `timeout` value is less than the `interval`. Default timeout is
  1s, which might be too short for some probes.
- Check network connectivity to the target from the host running Cloudprober.
- For HTTP probes, verify that the target URL is correct and accessible.

## BROWSER probes fail with "ERR_CERT_AUTHORITY_INVALID" against a custom CA

If a BROWSER probe's `page.goto` fails with `ERR_CERT_AUTHORITY_INVALID` when
the target is secured by a certificate from a custom (private) CA, adding the
certificate to the system trust store
(`/usr/local/share/ca-certificates/` + `update-ca-certificates`) or setting
`NODE_EXTRA_CA_CERTS` won't help — Chromium doesn't consult either. On Linux,
Chromium uses the NSS certificate database at `$HOME/.pki/nssdb` (`cert9.db`).

Build that database with `certutil`:

```shell
certutil -d sql:$HOME/.pki/nssdb -A -t "CT,C,C" -n "<CERT_NAME>" -i <CERT_PATH>
```

There are two pitfalls that produce this exact symptom:

1. **`$HOME` mismatch.** The NSS DB has to live under the `$HOME` of the user
   the container actually runs as when Chromium launches. If you build the DB
   as root but the container (or a k8s `runAsUser`) switches to a non-root uid
   at runtime, Chrome won't find it.
2. **Read-only root filesystem.** `cert9.db` is a SQLite database, and SQLite
   needs write access to its directory even just to open it for reads
   (journal/WAL files). If your deployment runs with a read-only root
   filesystem (common in k8s), an NSS DB baked into the image at
   `$HOME/.pki/nssdb` will fail to open at runtime even though `certutil`
   succeeded at build time. The fix is to copy the pre-built DB to a writable
   location (e.g. `/tmp/.pki/nssdb`) at container startup — before Cloudprober's
   Playwright subprocess launches — and point `$HOME` there.
