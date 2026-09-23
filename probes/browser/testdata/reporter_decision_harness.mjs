// reporter_decision_harness.mjs drives the rendered cloudprober Playwright
// reporter through synthetic reporter-event streams and checks, for each
// scenario, whether it emits the internal-error sentinel. It exercises the
// browserReady/onEnd decision without a real browser (which the CI/sandbox
// can't install), covering the launch-hang vs target-hang distinction that a
// Go-side regex test can't.
//
// Usage: node --experimental-strip-types reporter_decision_harness.mjs <reporter.ts>
// Exits 0 if every scenario matches its expected outcome, 1 otherwise.
//
// The event sequences mirror what real Playwright emits (confirmed against a
// real browser): the first Playwright API call is the browser launch, so a
// "pw:api" step that ends without error means the browser came up.

import { pathToFileURL } from "node:url";

const SENTINEL = "[cloudprober-internal-error]";

const reporterPath = process.argv[2];
if (!reporterPath) {
  console.error("usage: harness <reporter.ts>");
  process.exit(2);
}
// Convert to a file:// URL so dynamic import() accepts a Windows absolute path
// (e.g. C:\...), which node's ESM loader otherwise reads as a URL scheme.
const Reporter = (await import(pathToFileURL(reporterPath).href)).default;

// Event helpers. The browser launch is a Playwright API call ("pw:api" step):
// its onStepBegin models "launch started", and a matching onStepEnd without
// error models "browser came up". "hook" models non-Playwright work (e.g. a
// beforeAll doing a plain fetch). "end" is onEnd with the given suite status.
const apiBegin = { t: "stepBegin", category: "pw:api" };
const apiOk = { t: "stepEnd", category: "pw:api", error: undefined };
const apiErr = { t: "stepEnd", category: "pw:api", error: { message: "boom" } };
const hookBegin = { t: "stepBegin", category: "hook" };
const end = (status) => ({ t: "end", status });

const scenarios = [
  // Launch hang: launch started (pw:api begins) but never completed before the
  // global timeout. Internal error.
  { name: "launch_hang_timeout", events: [apiBegin, end("timedout")], wantSentinel: true },
  // Launch started then reported an error (no successful pw:api) then timeout.
  { name: "launch_errored_timeout", events: [apiBegin, apiErr, end("timedout")], wantSentinel: true },
  // Mid-test hang: browser launched (pw:api ok), a later navigation hung until
  // the global timeout. Target failure, not internal.
  { name: "midtest_hang_timeout", events: [apiBegin, apiOk, end("timedout")], wantSentinel: false },
  // Multi-test: an earlier test ran (pw:api ok), a later one was in flight at
  // the global timeout. Browser clearly worked -> target failure.
  { name: "multitest_hang_timeout", events: [apiBegin, apiOk, apiOk, end("timedout")], wantSentinel: false },
  // Pre-launch hang: a hook (e.g. beforeAll raw fetch) hung before the browser
  // was ever touched -- no pw:api step began. Target failure, not internal.
  { name: "prelaunch_hook_hang_timeout", events: [hookBegin, end("timedout")], wantSentinel: false },
  // Normal outcomes: not a global timeout at all.
  { name: "passed", events: [apiBegin, apiOk, end("passed")], wantSentinel: false },
  { name: "failed_not_timedout", events: [apiBegin, apiOk, end("failed")], wantSentinel: false },
  { name: "interrupted", events: [end("interrupted")], wantSentinel: false },
];

function runScenario(s) {
  const r = new Reporter();
  const origWrite = process.stderr.write.bind(process.stderr);
  let captured = "";
  process.stderr.write = (chunk) => { captured += chunk; return true; };
  try {
    for (const e of s.events) {
      const step = { category: e.category, error: e.error, title: "", duration: 0 };
      if (e.t === "stepBegin") {
        r.onStepBegin({}, {}, step);
      } else if (e.t === "stepEnd") {
        r.onStepEnd({}, {}, step);
      } else if (e.t === "end") {
        r.onEnd({ status: e.status });
      }
    }
  } finally {
    process.stderr.write = origWrite;
  }
  return captured.includes(SENTINEL);
}

let failures = 0;
for (const s of scenarios) {
  const got = runScenario(s);
  const ok = got === s.wantSentinel;
  if (!ok) failures++;
  console.log(`${ok ? "PASS" : "FAIL"}  ${s.name}: sentinel got=${got} want=${s.wantSentinel}`);
}
process.exit(failures === 0 ? 0 : 1);
