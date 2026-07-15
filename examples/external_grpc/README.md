# EXTERNAL_GRPC probe example

The EXTERNAL_GRPC probe delegates probe runs to a **sidecar** process over
gRPC. Cloudprober keeps scheduling, targets, metrics, validators, surfacers,
and alerting; the sidecar runs the actual probe logic — typically heavyweight
integrations that need real drivers (databases, message queues, etc.) — in
its own process, in any language, on its own release cadence.

This directory contains:

- `sidecar/` — an example sidecar built with the
  [`sidecar`](../../pkg/sidecar) SDK. It serves two probe types: `http`
  (stateful: per-target HTTP client cached across cycles) and `tcp`
  (stateless).
- `cloudprober.cfg` — a cloudprober config that probes real websites through
  the sidecar.

## Run it

Start the sidecar (listens on a unix socket by default):

```sh
go run ./sidecar
```

In another terminal, start cloudprober:

```sh
go run ../../cmd/cloudprober --config_file=cloudprober.cfg
```

Then look at the metrics:

```sh
curl -s localhost:9313/metrics | grep sidecar
```

You should see `total`, `success`, `latency`, and `internal_errors` for both
probes, plus the sidecar-provided `resp_bytes` metric for the `http` probe.
Sidecar/infra failures (e.g. sidecar down) count as probe failures like any
other — alerting on `total`/`success` keeps working — and additionally
increment `internal_errors`, so you can tell a broken sidecar apart from a
broken target; the failure reason is in cloudprober's logs. Try killing the
sidecar to see it in action.

## Serving over TLS / mTLS

The unix-socket default is meant for a co-located sidecar. To reach one over
the network, serve over TLS so probe configs (which may carry credentials)
aren't sent in the clear. The example takes cert flags:

```sh
go run ./sidecar --addr=:9314 \
  --tls_cert=server.crt --tls_key=server.key --client_ca=ca.crt
```

Passing `--client_ca` requires clients to present a cert signed by that CA
(mutual TLS), so only cloudprober can send probes. Omitting `--client_ca`
serves plain server-side TLS.

On the cloudprober side, set `tls_config` in the probe:

```
probe {
  name: "web_via_sidecar"
  type: EXTERNAL_GRPC
  targets { host_names: "cloudprober.org" }
  external_grpc_probe {
    server: "sidecar.example:9314"  # the sidecar, not the target
    probe_type: "http"
    tls_config {
      ca_cert_file: "ca.crt"      # verifies the sidecar's server cert
      tls_cert_file: "client.crt" # client cert (for mTLS)
      tls_key_file: "client.key"
      # server_name: "sidecar.example"
    }
  }
}
```

`server_name` overrides the name checked against the sidecar's server cert.
Set it when `server` is an IP or otherwise doesn't match a name/IP SAN on the
cert (gRPC otherwise derives the expected name from the dial target, and
verification fails) — e.g. `server: "10.0.0.5:9314"` with a cert issued for
`sidecar.example` needs `server_name: "sidecar.example"`. Avoid
`disable_cert_validation`: it skips verification entirely, which defeats the
point of protecting credential-bearing probe configs on the wire.

## Writing your own sidecar

A probe type is one struct — config in, result + metrics out; the SDK owns
the gRPC server, health service, session cache, and idle-TTL eviction. See
`sidecar/main.go` here and the package docs in
[`sidecar`](../../pkg/sidecar/sidecar.go).
