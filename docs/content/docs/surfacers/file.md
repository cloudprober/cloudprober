---
title: "File Surfacer"
menu:
  docs:
    parent: "surfacers"
    weight: 25
---

The file surfacer writes metrics as lines of text, either to a file or, if no
file path is configured, to standard output. It's enabled automatically, along
with the prometheus surfacer, if you don't configure any surfacer at all.

## Output format

Each line corresponds to one
[EventMetrics](https://pkg.go.dev/github.com/cloudprober/cloudprober/metrics#EventMetrics)
record and is built from four parts:

```
cloudprober 1500 1500590520 labels=ptype=http,probe=google-homepage,dst=google.com total=17 success=17 latency=180835
^^^^^^^^^^^ ^^^^ ^^^^^^^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
   prefix    id   timestamp                        labels                                       metrics
```

- **prefix** — a fixed string that makes the lines easy to grep for. It's
  `cloudprober` by default and is configurable with the `prefix` field.
- **id** — a counter seeded from the nanosecond timestamp at startup and
  incremented for every line this surfacer writes. It exists so that lines stay
  distinguishable on a shared stream such as a serial port; the only guarantee
  is that it goes up within one instance.
- **timestamp** — the time of the record, in Unix seconds.
- **labels** — comma separated `key=value` pairs, always prefixed with
  `labels=`. `ptype` is the probe type and `probe` is the probe name; the
  remaining labels depend on the probe and its targets.
- **metrics** — space separated `name=value` pairs.

## Metric values

Most metrics are plain numbers, but two kinds of values have a structure of
their own.

**Maps** are written as `map:<key-name>,<key>:<value>,...`, where the first
element after `map:` is the name of the map key. For example, HTTP response
codes appear as:

```
resp-code=map:code,200:44,204:8
```

meaning 44 responses with code 200 and 8 with code 204.

**Distributions** (latency histograms, when enabled) are written as four
sections separated by `|`:

```
latency=dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12
```

`sum` is the sum of all observed values, `count` the number of observations,
`lb` the lower bounds of the buckets, and `bc` the number of observations in
each bucket. Bucket `i` covers `lb[i]` up to `lb[i+1]`, and the last bucket is
unbounded, so the example above has 34 values below 0.5, 54 in [0.5, 2), 121 in
[2, 7.5) and 12 at 7.5 or above. `lb` and `bc` always have the same number of
elements.

String values are quoted, for example `hostname="prober-1"`.

Note that latency is reported in the unit configured by `latency_unit`
(microseconds by default), so the raw number doesn't tell you the unit on its
own.

## Sysvars lines

Lines with `ptype=sysvars` come from cloudprober itself rather than from a
probe, and carry process level variables such as `hostname`, `uptime` and
`version`:

```
cloudprober 1501 1500590530 labels=ptype=sysvars,probe=sysvars hostname="prober-1" uptime=100
```

## Configuration

See the
[file surfacer config](https://cloudprober.org/docs/config/surfacer/#cloudprober_surfacer_file_SurfacerConf)
for all the options, for example writing to a file instead of stdout, or
enabling compression.
