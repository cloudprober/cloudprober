// reporter_decision_harness.mjs drives the rendered cloudprober Playwright
// reporter through synthetic reporter-event streams and checks, for each
// scenario, whether it emits the internal-error sentinel and, optionally, the
// expected output lines (logs and step metrics). It exercises the
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
const testEnd = (status, errors) => ({ t: "testEnd", status, errors });

// test.step events. The same step object must be passed to its begin and end
// events (as Playwright does), so these take the step object.
const tstep = (title, error) => ({ category: "test.step", title, error, duration: 5 });
const stepBegin = (step) => ({ t: "stepBegin", step });
const stepEnd = (step) => ({ t: "stepEnd", step });
const stdout = (chunk) => ({ t: "stdout", chunk });
const stderr = (chunk) => ({ t: "stderr", chunk });

const okStep = tstep("ok step");
const badStep = tstep("bad step", { message: "\x1b[31mError: boom\x1b[39m\n  at foo.ts:1" });
const hungStep = tstep("hung step");

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

  // Step logs and metrics.
  {
    name: "step_status",
    events: [stepBegin(okStep), stepEnd(okStep), stepBegin(badStep), stepEnd(badStep), end("failed")],
    wantSentinel: false,
    wantOutput: [
      'DEBUG Test step "ok step" of test "t1" passed in 5ms',
      'test_step_status{step="ok_step",test="t1",status="passed"} 1',
      'WARNING Test step "bad step" of test "t1" failed: Error: boom\n',
      'test_step_status{step="bad_step",test="t1",status="failed"} 1',
    ],
  },
  // A step that never ended is reported, even if the run status is "failed"
  // (as it is when an earlier test failed) rather than "timedout".
  {
    name: "step_running_at_end",
    events: [stepBegin(okStep), stepEnd(okStep), stepBegin(hungStep), end("failed")],
    wantSentinel: false,
    wantOutput: ['WARNING Test step "hung step" of test "t1" was still running when the test run ended'],
  },
  // A failed test logs only its error messages, without ANSI codes, the
  // stack or the code snippet.
  {
    name: "test_failed_errors",
    events: [testEnd("failed", [{
      message: "\x1b[31mError: page.goto: net::ERR_NAME_NOT_RESOLVED\x1b[39m\nCall log:\n  - navigating to \"https://example.internal/\"",
      stack: "Error: page.goto: ...\n    at /tests/a.spec.ts:5:14",
      snippet: "\x1b[31m> 5 |\x1b[39m await page.goto(url);",
      location: { file: "/tests/a.spec.ts", line: 5, column: 14 },
    }, { value: "thrown string" }])],
    wantSentinel: false,
    wantOutput: [
      'WARNING Test "t1" failed with errors: ["Error: page.goto: net::ERR_NAME_NOT_RESOLVED\\nCall log:\\n  - navigating to \\"https://example.internal/\\" (at /tests/a.spec.ts:5)","thrown string"]\n',
    ],
  },
  // Worker output: playwright debug logs at DEBUG, the rest at the stream's
  // level, and long lines truncated.
  {
    name: "worker_output",
    events: [
      stdout("hello\n\x1b[38;5;45;1m  pw:api \x1b[0m=> page.goto started\n"),
      stderr("oops\n"),
      stdout("x".repeat(10000) + "\n"),
    ],
    wantSentinel: false,
    wantOutput: [
      "INFO [t1] hello\n",
      "DEBUG [t1]   pw:api => page.goto started\n",
      "WARNING [t1] oops\n",
      "INFO [t1] " + "x".repeat(8192) + "... (truncated)\n",
    ],
  },
];

const test = { title: "t1", parent: { title: "" }, tags: [] };

function runScenario(s) {
  const r = new Reporter();
  const origStderr = process.stderr.write.bind(process.stderr);
  const origStdout = process.stdout.write.bind(process.stdout);
  let captured = "";
  process.stderr.write = (chunk) => { captured += chunk; return true; };
  process.stdout.write = (chunk) => { captured += chunk; return true; };
  try {
    for (const e of s.events) {
      const step = e.step ?? { category: e.category, error: e.error, title: "", duration: 0 };
      if (e.t === "stepBegin") {
        r.onStepBegin(test, {}, step);
      } else if (e.t === "stepEnd") {
        r.onStepEnd(test, {}, step);
      } else if (e.t === "stdout") {
        r.onStdOut(e.chunk, test, {});
      } else if (e.t === "stderr") {
        r.onStdErr(e.chunk, test, {});
      } else if (e.t === "testEnd") {
        r.onTestEnd(test, { status: e.status, errors: e.errors, duration: 0 });
      } else if (e.t === "end") {
        r.onEnd({ status: e.status });
      }
    }
  } finally {
    process.stderr.write = origStderr;
    process.stdout.write = origStdout;
  }
  return captured;
}

let failures = 0;
for (const s of scenarios) {
  const captured = runScenario(s);
  const got = captured.includes(SENTINEL);
  const missing = (s.wantOutput ?? []).filter((want) => !captured.includes(want));
  const ok = got === s.wantSentinel && missing.length === 0;
  if (!ok) failures++;
  console.log(`${ok ? "PASS" : "FAIL"}  ${s.name}: sentinel got=${got} want=${s.wantSentinel}`);
  for (const want of missing) {
    console.log(`      missing output: ${JSON.stringify(want)}\n      got: ${JSON.stringify(captured)}`);
  }
}
process.exit(failures === 0 ? 0 : 1);
