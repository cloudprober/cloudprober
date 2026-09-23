# Dual-stack (IPv4 + IPv6) probing

IPv4 and IPv6 targets cannot be mixed within a single probe. Packet-level
probes (PING, UDP) need to know the address family upfront to craft packets,
and default to IPv4 when `ip_version` is not set. To monitor both families,
run one probe per family with `ip_version` set explicitly on each.

[`cloudprober.cfg`](cloudprober.cfg) pings two hosts over both families.

## Keeping metrics comparable across families

The example declares targets with `endpoint` (name + `ip`) rather than
`host_names`. The `dst` metric label comes from the endpoint name, so the same
host is directly comparable across the two probes:

```
total{ptype="ping",probe="net4",dst="serverA"} 20
total{ptype="ping",probe="net6",dst="serverA"} 20
```

With `host_names`, `dst` would carry the raw address instead, and the v4 and v6
series for a host would have nothing in common to join on.

`endpoint` also accepts `port`, `url`, and `labels` -- see the
[targets documentation](https://cloudprober.org/docs/how-to/targets/) for the
full set.
