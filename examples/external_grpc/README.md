# EXTERNAL_GRPC probe example

The EXTERNAL_GRPC probe delegates probe runs to a **sidecar** process over
gRPC. Cloudprober keeps scheduling, targets, metrics, validators, surfacers,
and alerting; the sidecar runs the actual probe logic — typically heavyweight
integrations that need real drivers (databases, message queues, etc.) — in
its own process, in any language, on its own release cadence.

This directory contains:

- `sidecar/` — an example sidecar built with the
  [`sidecar`](../../sidecar) SDK. It serves two probe types: `http`
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
`internal_errors` counts sidecar/infra failures (e.g. sidecar down) —
separately from probe failures, so a broken sidecar doesn't look like a
broken target. Try killing the sidecar to see it in action.

## Writing your own sidecar

A probe type is one struct — config in, result + metrics out; the SDK owns
the gRPC server, health service, session cache, and idle-TTL eviction. See
`sidecar/main.go` here and the package docs in
[`sidecar`](../../sidecar/sidecar.go).
