# Browser Probe: Getting Started

Cloudprober's Browser Probe brings **continuous end-to-end monitoring** to your
web applications. It runs real browser interactions -- navigating pages, clicking
buttons, filling forms -- on a schedule, and feeds the results into
Cloudprober's observability pipeline. If your checkout flow breaks at 3 AM,
you'll know in seconds, not when the first customer complaint arrives.

The probe uses Playwright under the hood, so you can leverage existing Playwright
tests and the broader ecosystem. But the real value is what Cloudprober adds on
top: scheduling, metrics, artifact management, retries, and multi-target
support -- everything needed to make browser tests operate as production
monitoring.

**TL;DR:** Cloudprober makes Playwright production-ready.

---

## Why Not Just Run Playwright on Its Own?

A standalone Playwright test tells you *pass* or *fail* at the moment you run it.
That's fine in CI. In production, you need continuous execution, time-series
metrics, alerting, and a way to investigate failures after the fact. Building that
yourself means cron jobs, custom metric exporters, artifact storage scripts, and
dashboard plumbing. The Browser Probe handles all of it as a first-class
Cloudprober citizen.

### Automated Observability -- Zero Instrumentation

Every Browser Probe run automatically emits:

| Metric | Description |
|---|---|
| `total` / `success` / `latency` | Standard Cloudprober probe-level counters |
| `test_status{test, suite, tags, status}` | Per-test pass/fail counter |
| `test_latency{test, suite, tags, status}` | Per-test duration (microseconds) |

These metrics flow directly to any configured surfacer -- **Prometheus,
Stackdriver (Google Cloud Monitoring), OpenTelemetry, CloudWatch**, and more --
with no extra configuration. Point your existing alerting at
`success < total` and you have end-to-end monitoring.

#### Per-Step Custom Metrics

Need to know how long the "Add to cart" step takes *inside* a checkout test?
Enable step-level metrics:

```textproto
test_metrics_options {
  enable_step_metrics: true
}
```

This adds two more metric families:

- `test_step_status{step, test, suite, tags, status}`
- `test_step_latency{step, test, suite, tags, status}`

In your Playwright test, wrap operations in `test.step()` and the reporter picks
them up automatically (see the test example below).

### Artifacts Management -- Built-in Viewer UI

Screenshots, traces, and HTML reports are generated on every run. Cloudprober
stores them, cleans up old ones, and serves them through a built-in **Artifacts
Viewer** at `/artifacts/<probe_name>/`.

The viewer lets you:

- **Browse by date and timestamp** -- artifacts are organized as
  `YYYY-MM-DD/<unix_ms>/<target>/`.
- **Filter by time range** -- datetime pickers for start/end.
- **Filter by failure** -- a "Failure Only" checkbox highlights runs where the
  probe failed (indicated by a `cloudprober_probe_failed` marker file).
- **Drill down** -- click through to the Playwright HTML report, individual
  screenshots, or trace files.

Artifacts can be stored locally *and* pushed to **S3, GCS, or Azure Blob
Storage** simultaneously -- Cloudprober uploads to each backend in parallel
without blocking the probe.

### Operational Niceties

- **Retries with trace capture** -- `save_trace: ON_FIRST_RETRY` captures a
  Playwright trace only when a test is retried, keeping storage lean.
- **Automatic cleanup** -- old artifacts are garbage-collected based on
  `max_age_sec`.
- **Target-aware** -- environment variables like `target_name`, `target_ip`,
  `target_port`, and `target_label_*` are injected into every test run, so a
  single test spec can probe multiple endpoints.

---

## Quick Start

### 1. Cloudprober Config

```textproto
probe {
  name: "website_e2e"
  type: BROWSER
  interval_msec: 60000
  timeout_msec: 30000

  targets {
    host_names: "www.example.com"
  }

  browser_probe {
    test_spec: "checkout.spec.ts"
    test_dir: "/tests"
    retries: 1
    save_trace: RETAIN_ON_FAILURE

    test_metrics_options {
      enable_step_metrics: true
    }

    artifacts_options {
      serve_on_web: true
      storage {
        local_storage {
          dir: "/artifacts"
          cleanup_options {
            max_age_sec: 86400
          }
        }
      }
    }
  }
}
```

This runs `checkout.spec.ts` once a minute against `www.example.com`, retries
failures once, captures traces on failure, exports per-step metrics, and stores
artifacts locally for 24 hours.

### 2. Sample Playwright Test (TypeScript)

```typescript
// /tests/checkout.spec.ts
import { test, expect } from "@playwright/test";

// target_name is injected by Cloudprober as an env var.
const baseURL = `https://${process.env.target_name}`;

test.describe("Checkout Flow", () => {
  test("complete purchase", async ({ page }) => {
    await test.step("Navigate to store", async () => {
      await page.goto(baseURL);
      await expect(page).toHaveTitle(/Store/);
    });

    await test.step("Add to cart", async () => {
      await page.getByRole("button", { name: "Add to cart" }).click();
      await expect(page.getByTestId("cart-count")).toHaveText("1");
    });

    await test.step("Complete checkout", async () => {
      await page.getByRole("link", { name: "Checkout" }).click();
      await page.getByPlaceholder("Card number").fill("4111111111111111");
      await page.getByRole("button", { name: "Pay" }).click();
      await expect(page.getByText("Order confirmed")).toBeVisible();
    });
  });
});
```

Each `test.step()` call becomes a `test_step_status` /
`test_step_latency` metric automatically. No SDK, no custom reporter wiring --
Cloudprober generates the Playwright config and reporter at startup.

---

## Configuration Reference

All fields from `probes/browser/proto/config.proto`:

| Field | Type | Default | Description |
|---|---|---|---|
| `test_spec` | `repeated string` | *(all files in test_dir)* | Test specs to run. Filenames are resolved relative to `test_dir`; strings containing regex characters (`^$*\|?+()[]{}`) are passed as Playwright grep patterns. |
| `test_dir` | `string` | Config file directory | Directory where test specs are located. |
| `test_spec_filter` | `TestSpecFilter` | -- | Include/exclude tests by title regex. Maps to Playwright `--grep` / `--grep-invert`. |
| `playwright_dir` | `string` | `$PLAYWRIGHT_DIR` | Path to Playwright installation. Auto-set in the official `cloudprober:*-pw` images. |
| `npx_path` | `string` | `"npx"` | Path to the `npx` binary. |
| `workdir` | `string` | *(auto temp dir)* | Writable working directory. Leave unset unless you need persistence. |
| `retries` | `int32` | `0` | Number of retries per test. |
| `save_screenshots_for_success` | `bool` | `false` | Save screenshots for passing tests (failures always capture screenshots). |
| `save_trace` | `SaveOption` | `NEVER` | Trace capture strategy: `NEVER`, `ALWAYS`, `ON_FIRST_RETRY`, `ON_ALL_RETRIES`, `RETAIN_ON_FAILURE`. |
| `test_metrics_options` | `TestMetricsOptions` | -- | `disable_test_metrics`, `disable_aggregation`, `enable_step_metrics`. |
| `artifacts_options` | `ArtifactsOptions` | *(global if set)* | Per-probe artifact storage and web serving config. Falls back to `global_artifacts_options`. |
| `workdir_cleanup_options` | `CleanupOptions` | 1hr max age | Cleanup schedule for the working directory. |
| `env_var` | `map<string,string>` | -- | Extra environment variables passed to the Playwright process. |
| `requests_per_probe` | `int32` | `1` | Concurrent Playwright invocations per probe cycle. |
| `requests_interval_msec` | `int32` | `0` | Stagger delay between concurrent requests. |

**`TestSpecFilter`**

```textproto
test_spec_filter {
  include: "@smoke|@critical"   # playwright --grep
  exclude: "@draft"             # playwright --grep-invert
}
```

**`SaveOption` enum values:**
`NEVER` | `ALWAYS` | `ON_FIRST_RETRY` | `ON_ALL_RETRIES` | `RETAIN_ON_FAILURE`

---

## Artifacts Setup

### Enable the Built-in Artifacts Server

The artifacts viewer is served by Cloudprober's built-in web server but is
**disabled by default** for security. To enable it:

#### Option A: Global Config (Recommended)

Set `global_artifacts_options` at the top level of your Cloudprober config. This
applies to all browser probes and keeps per-probe configs clean.

```textproto
global_artifacts_options {
  serve_on_web: true
  storage {
    local_storage {
      dir: "/artifacts"
      cleanup_options {
        max_age_sec: 86400          # 24 hours
        cleanup_interval_sec: 3600
      }
    }
  }
}
```

#### Option B: Per-Probe Config

Set `artifacts_options` inside the `browser_probe` block (as shown in the Quick
Start above). Per-probe options override the global config.

### Adding Cloud Storage Backends

Storage backends are defined in the `storage` field. You can specify multiple
backends -- Cloudprober uploads to all of them in parallel.

```textproto
global_artifacts_options {
  serve_on_web: true

  # Local -- required for the web viewer
  storage {
    local_storage {
      dir: "/artifacts"
      cleanup_options {
        max_age_sec: 86400
      }
    }
  }

  # S3 -- long-term archival
  storage {
    s3 {
      bucket: "my-monitoring-artifacts"
      region: "us-west-2"
    }
    path: "browser-probes"
  }

  # GCS -- alternative cloud backend
  storage {
    gcs {
      bucket: "my-gcs-bucket"
    }
    path: "cloudprober/artifacts"
  }
}
```

Azure Blob Storage (`abs`) is also supported with shared-key or managed-identity
auth.

### Accessing the Viewer

Once `serve_on_web: true` is set and Cloudprober is running, open:

```
http://<cloudprober-host>:9313/artifacts/<probe_name>/
```

The default Cloudprober port is `9313`. The viewer shows a date-grouped index of
all stored runs. Use the datetime pickers and "Failure Only" checkbox to narrow
results during incident triage.

For a raw file tree (useful for scripting / direct links), use:

```
http://<cloudprober-host>:9313/artifacts/<probe_name>/tree/
```

---

## Running with Docker

The official Cloudprober images with the `-pw` tag suffix come with Playwright
and Chromium pre-installed:

```bash
docker run --rm -v /path/to/config.cfg:/etc/cloudprober.cfg \
  -v /path/to/tests:/tests \
  -v /tmp/artifacts:/artifacts \
  -p 9313:9313 \
  cloudprober/cloudprober:latest-pw
```

The `$PLAYWRIGHT_DIR` environment variable is pre-configured in these images, so
you don't need to set `playwright_dir` in your config.
