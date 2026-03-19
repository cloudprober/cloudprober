---
menu:
  docs:
    parent: "faq"
    weight: 20
title: "Alerting"
date: 2026-03-15
---

## Does Cloudprober support alerting?

Yes. Cloudprober has built-in alerting that can notify you via Email (SMTP),
PagerDuty, Opsgenie, Slack, generic HTTP webhooks, or shell commands. See the
[Alerting](/docs/how-to/alerting) guide for setup instructions.

## How does alert condition work?

Alerts are based on failure thresholds. You configure the number of failures
within a window of total probes. For example, "alert if 2 out of the last 10
probes fail":

```shell
alerting {
  alert_condition {
    failures: 2
    total: 10
  }
}
```

## Can I customize alert messages?

Yes. Alert notifications support templates with placeholders such as
`@probe@`, `@target@`, `@failures@`, and `@total@` that get replaced with
actual values at alert time.

## Do I get notified when an alert resolves?

Yes. Cloudprober sends a resolved notification when the alert condition is no
longer met, so you know when things are back to normal.
