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

If a BROWSER probe fails with `ERR_CERT_AUTHORITY_INVALID` on a target secured
by a custom (private) CA, the usual fixes don't apply. Chromium on Linux
ignores both the system trust store (`/usr/local/share/ca-certificates/` +
`update-ca-certificates`) and `NODE_EXTRA_CA_CERTS`; it reads certificates from
the NSS database at `$HOME/.pki/nssdb` (`cert9.db`) instead.

Import your CA into that database with `certutil`:

```shell
mkdir -p $HOME/.pki/nssdb && \
  certutil -d sql:$HOME/.pki/nssdb -A -t "CT,C,C" -n "<CERT_NAME>" -i <CERT_PATH>
```

In containerized deployments, two things commonly make this fail silently:

1. **`$HOME` mismatch.** The database must live under the `$HOME` of the user
   Chromium actually runs as. Build it as root but let the container drop to a
   non-root uid at runtime (e.g. k8s `runAsUser`), and Chromium won't find it.
2. **Read-only root filesystem.** `cert9.db` is a SQLite file, and SQLite must
   create journal files in the directory containing the database in order to
   write to it. Chromium's NSS library opens the database in read-write mode,
   so that directory has to be writable. A database baked into the image at
   `$HOME/.pki/nssdb` therefore can't be used when the root filesystem is
   read-only (common in k8s), even though `certutil` succeeded at build time.

The fix for the read-only case is to copy the pre-built database to a writable
location at startup and point `$HOME` there. (Build it during the Docker build
with `certutil` — the runtime image doesn't ship `certutil`.) For example, if
your image has the certs database at `/nssdb`, readable by the uid the container
runs as (k8s `runAsUser`), you can configure your container command as the
following to make chrome use your certs:

```yaml
command:
  - /bin/sh
  - -c
  - mkdir -p /tmp/node-home/.pki/nssdb &&
    cp -f /nssdb/* /tmp/node-home/.pki/nssdb/ &&
    HOME=/tmp/node-home exec /cloudprober "$@"
  - --
```

If you don't build your own Cloudprober image, you can do all of this in an
init container instead. Run an image that has `certutil` (Alpine's `nss-tools`,
Debian's `libnss3-tools`), build the database into a shared `emptyDir`, and
point `$HOME` at it. The volume is writable, so the read-only root filesystem
problem goes away and there's no need to override the container `command`:

```yaml
volumes:
  - name: nss-home
    emptyDir: {}
  - name: ca
    configMap:
      name: custom-ca # contains ca.crt

initContainers:
  - name: build-nssdb
    image: <image-with-certutil>
    command:
      - /bin/sh
      - -c
      - mkdir -p /nss-home/.pki/nssdb &&
        certutil -d sql:/nss-home/.pki/nssdb -A -t "CT,C,C"
        -n custom-ca -i /ca/ca.crt
    volumeMounts:
      - name: nss-home
        mountPath: /nss-home
      - name: ca
        mountPath: /ca

containers:
  - name: cloudprober
    image: cloudprober/cloudprober:latest-pw
    env:
      - name: HOME
        value: /nss-home
    volumeMounts:
      - name: nss-home
        mountPath: /nss-home
```

If the main container runs as a non-root user, run the init container as that
same uid (e.g. set `runAsUser` at the pod level) so the database it creates is
readable and writable by the main container.
