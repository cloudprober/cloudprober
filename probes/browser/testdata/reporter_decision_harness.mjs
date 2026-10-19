// reporter_decision_harness.mjs drives the rendered cloudprober Playwright
// reporter through synthetic reporter-event streams and checks, for each
// scenario, whether it emits the internal-error sentinel (stderr) and/or the
// expected stdout metric line. It exercises the browserReady/onEnd decision
// and the per-step metric labels without a real browser (which the
// CI/sandbox can't install) -- the launch-hang vs target-hang distinction and
// the step status label, neither of which a Go-side regex test can check.
//
// Usage: node --experimental-strip-types reporter_decision_harness.mjs <reporter.ts>
// Exits 0 if every scenario matches its expected outcome, 1 otherwise.
//
// The event sequences mirror what real Playwright emits (confirmed against a
// real browser): the first Playwright API call is the browser launch, so a
// "pw:api" step that ends without error means the browser came up. Likewise,
// during onStepEnd the TestResult passed in is the test's own result, still
// at its unfinalized default status ("skipped") -- it's only finalized later,
// in onTestEnd. A "test.step" step has no status of its own in the Playwright
// API; only step.error says whether it failed.

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

// A "test.step" step, as reported during onStepEnd: test/result are the real
// (still in-progress) objects testLabels() and the status label read from,
// not the placeholder {} used by the pw:api scenarios above.
const stepTest = { title: "example test", parent: { title: "" }, tags: [] };
const stepResult = { status: "skipped" };
const appStepOk = {
  t: "stepEnd", category: "test.step", error: undefined,
  test: stepTest, result: stepResult, title: "ok_step",
};
const appStepErr = {
  t: "stepEnd", category: "test.step", error: { message: "boom" },
  test: stepTest, result: stepResult, title: "err_step",
};

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
  // test_step_status/test_step_latency must reflect the step's own error, not
  // the enclosing test's still-unfinalized ("skipped") result.status.
  {
    name: "step_without_error_reports_passed", events: [appStepOk],
    wantSentinel: false, wantStdout: 'test_step_status{step="ok_step",test="example_test",status="passed"}',
  },
  {
    name: "step_with_error_reports_failed", events: [appStepErr],
    wantSentinel: false, wantStdout: 'test_step_status{step="err_step",test="example_test",status="failed"}',
  },
];

function runScenario(s) {
  const r = new Reporter();
  const origErrWrite = process.stderr.write.bind(process.stderr);
  const origOutWrite = process.stdout.write.bind(process.stdout);
  let capturedErr = "";
  let capturedOut = "";
  process.stderr.write = (chunk) => { capturedErr += chunk; return true; };
  process.stdout.write = (chunk) => { capturedOut += chunk; return true; };
  try {
    for (const e of s.events) {
      const step = { category: e.category, error: e.error, title: e.title ?? "", duration: 0 };
      const test = e.test ?? {};
      const result = e.result ?? {};
      if (e.t === "stepBegin") {
        r.onStepBegin(test, result, step);
      } else if (e.t === "stepEnd") {
        r.onStepEnd(test, result, step);
      } else if (e.t === "end") {
        r.onEnd({ status: e.status });
      }
    }
  } finally {
    process.stderr.write = origErrWrite;
    process.stdout.write = origOutWrite;
  }
  return { sentinel: capturedErr.includes(SENTINEL), stdout: capturedOut };
}

let failures = 0;
for (const s of scenarios) {
  const got = runScenario(s);
  let ok = got.sentinel === s.wantSentinel;
  if (ok && s.wantStdout !== undefined) {
    ok = got.stdout.includes(s.wantStdout);
  }
  if (!ok) failures++;
  console.log(`${ok ? "PASS" : "FAIL"}  ${s.name}: sentinel got=${got.sentinel} want=${s.wantSentinel}`);
}
process.exit(failures === 0 ? 0 : 1);
