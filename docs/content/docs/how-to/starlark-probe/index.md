---
menu:
  docs:
    parent: "how-to"
    name: "Script Probe (Starlark)"
    weight: 14
title: "Script Probe (Starlark)"
date: 2026-05-10T10:00:00-07:00
keywords: ["script probe", "starlark probe", "scripted probe", "multi-step", "api flow", "chained requests"]
---

Cloudprober's **script probe** (proto type `STARLARK`) lets you write
**multi-step checks as a small script**. The script is written in
[Starlark](https://github.com/bazelbuild/starlark) -- a sandboxed, Python-like
language with `if`/`for`/dicts/lists/string formatting -- and runs in-process,
once per resolved target each interval. Wall time of the call becomes the
probe's latency; a clean return is success, any unhandled error (including
assertion failures) is failure.

Think of it as **"curl + jq + assertions, in a real scripting language"**,
without the deployment baggage of shipping a binary alongside Cloudprober. A
typical use case is a chained API flow: get a token, list resources, fetch a
detail page, validate the response, emit a custom latency metric for the
critical step.

> **Why Starlark?** It's the sandboxed Python-like language behind Bazel and
> Buck. It gives you most of Python's ergonomics (string formatting, dicts,
> comprehensions) with strong guarantees: no filesystem, no `exec`, no module
> system, deterministic semantics, and cheap to embed -- perfect for short
> scripts that ship inside a config file.

## When to use it

| If you want to... | Use |
|---|---|
| Probe a single URL | [HTTP probe](https://cloudprober.org/docs/config/latest/probes/#cloudprober_probes_http_ProbeConf) |
| Run an arbitrary external binary, in any language | [External probe]({{< ref "external-probe" >}}) |
| Drive a real browser through user flows | [Browser probe]({{< ref "browser-probe" >}}) |
| Chain a few HTTP requests, assert on responses, emit custom metrics | **Script probe** (this page) |

The script probe sits in the gap between the HTTP probe (which is one request,
no logic) and the external probe (which is a whole binary you have to build,
ship, and version). For most "test an API end-to-end" checks, that gap is
exactly where you want to be.

## Quick Start

A small token-auth API check (full example in
[examples/starlark/](https://github.com/cloudprober/cloudprober/tree/master/examples/starlark)):

```python
# trading_api.star
def probe(target):
    base = "http://%s:%d" % (target.name, target.port)

    # 1. Auth.
    r = http.post(
        url = base + "/api-token-auth/",
        json = {"username": "demo", "password": "demo"},
    )
    assert.http_status(r, 200)
    token = r.json()["token"]
    auth = {"Authorization": "Token " + token}

    # 2. Accounts.
    r = http.get(url = base + "/accounts/", headers = auth)
    assert.http_status(r, 200)
    accounts = r.json()["results"]
    if len(accounts) == 0:
        fail("no accounts returned")

    # 3. Portfolio for first account.
    account = accounts[0]["account_number"]
    r = http.get(url = base + "/portfolios/%s/" % account, headers = auth)
    assert.http_status(r, 200)
    print("equity for %s: %s" % (account, r.json()["equity"]))
```

```proto
# cloudprober.cfg
probe {
  name: "trading_api_flow"
  type: STARLARK
  targets {
    host_names: "127.0.0.1:8080"
  }
  interval_msec: 5000
  timeout_msec: 3000
  starlark_probe {
    source_file: "trading_api.star"
  }
}
```

Cloudprober calls `probe(target)` every 5 seconds. Each `assert.http_status` raises
on mismatch, ending the run as a failure -- without any extra plumbing,
`total`, `success`, and `latency` metrics flow to your surfacers exactly like
any other probe.

## The `target` argument

The entry point receives a single argument exposing fields from the resolved
target endpoint:

| Attribute | Type | Notes |
|---|---|---|
| `target.name` | string | Host name or IP from discovery. |
| `target.port` | int | `0` if no port. |
| `target.ip` | string | Resolved IP (`""` if not yet resolved). |
| `target.labels` | dict[str,str] | Frozen; lookups via `target.labels.get("env", "prod")`. |

## Available builtins

All scripts get a small, fixed set of builtins. No filesystem, network beyond
`http`, or `exec` -- the probe is sandboxed by Starlark.

| Builtin | Purpose |
|---|---|
| `http.get(url, headers=None, max_redirects=N, keep_alive=False)` | HTTP GET, returns a `Response`. `max_redirects=0` disables following; `max_redirects=N` follows up to N (omit for Go's default of 10). `keep_alive` defaults to `False` to match the HTTP probe -- every call exercises DNS/TCP/TLS setup, which is what you usually want from a prober. Pass `keep_alive=True` to reuse a pooled connection across calls in a chained API flow. |
| `http.post(url, headers=None, body=None, json=None, max_redirects=N, keep_alive=False)` | HTTP POST. Pass `json=` for an auto-encoded JSON body (sets `Content-Type`), or `body=` for a raw string/bytes. `max_redirects` and `keep_alive` match `http.get`. |
| `assert.http_status(response, expected)` | Fails the probe if `response.status != expected`. Stamp per-call context onto the failure log line with `log.set_attr` instead of inlining it into the error message. |
| `vars.get(name, default=None)` | Read values from the probe's `vars` config map (see below). |
| `state.get(key, default=None)` / `state.set(key, value)` | Per-target key-value store that persists across runs (see below). |
| `oauth.token(name="")` / `oauth.header(name="")` | Fetch a token from one of the probe's configured `oauth_configs`. `token()` returns the raw access token; `header()` returns the formatted `Authorization` value (e.g. `Bearer <token>`). `name` is optional when exactly one config is set (see below). |
| `log.info(msg)` / `log.warn(msg)` / `log.error(msg)` / `log.debug(msg)` | Route a message to Cloudprober's logger with the probe's `target` attribute attached. |
| `log.set_attr(key, value)` | Add a sticky attribute to this run's logger. All subsequent `log.*` calls *and* the probe-failure log line Cloudprober writes if the script errors out (network timeout, assertion failure, etc.) carry it. Cleared at the end of the run. Useful for stamping `req_id` or other per-run context onto failure logs. |
| `print_metric(line)` | Emit a custom metric line (see below). |
| `print(...)` | Standard Starlark `print`; routed to the logger at INFO. |
| `fail(msg)` | Standard Starlark; ends the run as a failure. |

> `keep_alive=False` controls *post-request* pooling -- it doesn't drain
> an already-pooled connection. If you mix `True` and `False` calls to
> the same host, a `False` call may ride a previously-pooled connection
> before closing it.

### Response

```python
r = http.get(url = "https://example.com/")
r.status     # int, e.g. 200
r.headers    # dict[str,str]; multi-valued headers are ", "-joined
r.body       # bytes
r.json()     # parsed JSON (raises on parse error)
```

### TLS configuration

For HTTPS calls to hosts that need a custom CA bundle, mTLS client cert, SNI
override, or disabled cert validation, set `tls_config` on the probe. Same
message the HTTP probe uses; applies to every `http.get`/`http.post` in the
script. Example using a custom CA:

```proto
probe {
  type: STARLARK
  starlark_probe {
    source_file: "internal_api.star"
    tls_config {
      ca_cert_file: "/etc/ssl/internal-root-ca.pem"
    }
  }
}
```

For development against self-signed certs:

```proto
tls_config {
  disable_cert_validation: true
}
```

### `vars`

`vars.get` reads from the probe's static config map -- useful for passing
non-secret config (an environment name, an API base URL, a feature flag)
without rebuilding the script:

```proto
probe {
  type: STARLARK
  starlark_probe {
    source_file: "checkout.star"
    vars { key: "api_base" value: "https://api.staging.example.com" }
    vars { key: "feature_flag" value: "fast_checkout" }
  }
}
```

```python
def probe(target):
    base = vars.get("api_base", "https://api.example.com")
    ...
```

For host environment values, use Cloudprober's config-loading template layer
(e.g. `vars { key: "api_key" value: "{{ env "API_KEY" }}" }`).

### `state`

`state` is a per-`(probe, target)` dictionary that survives across runs. Use
it for things like remembering the last-seen ETag, a paging cursor, or a
rate-limit countdown:

```python
def probe(target):
    last_id = state.get("last_id", 0)
    r = http.get(url = "http://%s/events?since=%d" % (target.name, last_id))
    assert.http_status(r, 200)
    events = r.json()["events"]
    if events:
        state.set("last_id", events[-1]["id"])
```

The bucket lives only as long as the target does: when a target disappears
from discovery, its state is dropped. Each bucket holds up to 1024 keys.

### `oauth`

Configure one or more OAuth token sources on the probe with `oauth_configs`, a
map keyed by a name the script uses to select one. This is the same
`oauth.Config` message the HTTP probe uses -- token file, command, HTTP token
endpoint, GCE metadata, k8s token file, Google credentials, self-signed JWT --
with refresh and caching handled for you. See the
[OAuth]({{< ref "oauth" >}}) page for the full config surface.

Two builtins read these token sources:

- `oauth.token(name="")` returns the raw access token (`"eyJ..."`), for
  query-param tokens, signed-URL flows, or non-`Bearer` schemes.
- `oauth.header(name="")` returns the formatted `Authorization` value
  (`"Bearer eyJ..."`, per the config's `token_type_format`) -- the common case.

Unlike the HTTP probe, **tokens are never auto-injected**. A script routinely
hits several hosts in one run (an authenticated API, a public health endpoint,
a redirect target), and silently attaching the bearer token to all of them
would leak it to the wrong hosts. So the script asks for the token explicitly
and decides where it goes. A token-fetch failure surfaces as a script error,
failing the run.

`name` may be omitted when exactly one `oauth_config` is set; with several, pass
the name (omitting it, or naming an unknown config, is an error).

```proto
probe {
  type: STARLARK
  starlark_probe {
    source_file: "api_check.star"
    oauth_configs {
      key: "api"
      value {
        http_request {
          token_url: "https://issuer.example.com/oauth2/token"
          data: "grant_type=client_credentials"
          # ...
        }
      }
    }
  }
}
```

```python
def probe(target):
    r = http.get(
        url = "https://api.example.com/v1/me",
        headers = {"Authorization": oauth.header("api")},
    )
    assert.http_status(r, 200)

    h = http.get(url = "https://api.example.com/healthz")  # unauthenticated
    assert.http_status(h, 200)
```

## Custom metrics with `print_metric`

`print_metric(line)` accepts the same payload-format strings as the external
probe's stdout protocol. The simplest form is `name value`:

```python
print_metric("items_in_cart 5")
print_metric('checkout_latency_ms{flow="guest"} 234.5')
```

Configure how those lines are interpreted -- gauge vs. cumulative,
distribution buckets, in-Cloudprober aggregation -- with
`output_metrics_options` on the probe:

```proto
probe {
  type: STARLARK
  starlark_probe {
    source_file: "checkout.star"
    output_metrics_options {
      aggregate_in_cloudprober: true
      dist_metric {
        key: "checkout_latency_ms"
        value {
          explicit_buckets: "10,50,100,250,500,1000,5000"
        }
      }
    }
  }
}
```

The configuration knobs (kind, additional labels, JSON / header metrics,
distribution buckets) are exactly those used by the [external probe
output_metrics_options]({{< ref "external-probe#distributions" >}}); see that
section for a worked example.

Metrics are dispatched **as the line is emitted** -- if a later assertion
fails, every `print_metric` line before it still surfaces.

## Configuration reference

See the [generated config reference](https://cloudprober.org/docs/config/latest/probes/#cloudprober_probes_starlark_ProbeConf)
for the full schema. The most-used fields:

| Field | Default | Description |
|---|---|---|
| `source` / `source_file` | -- | Exactly one of these. Inline source or path to a `.star` file. |
| `entry_point` | `"probe"` | Function to call each run. Must take one argument (`target`). |
| `vars` | -- | `map<string,string>` exposed to the script via `vars.get`. |
| `output_metrics_options` | -- | How `print_metric` lines are parsed -- kind, distributions, aggregation, etc. |

## Lifecycle and concurrency

- Module-level code (top-level `def`, assignments, imports) runs **once** at
  probe load. Helper functions and constants defined there are reused on every
  call.
- The entry-point function is called once per target per interval. Each call
  runs on its own Starlark thread; **module globals are frozen** after load, so
  you cannot accumulate cross-run state in a module-level dict -- use `state`
  instead.
- `probe_timeout` bounds each call: an in-flight `http.get` is cancelled, and
  pure-Starlark loops are interrupted at the next call/branch.
