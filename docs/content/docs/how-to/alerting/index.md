---
menu:
  docs:
    parent: how-to
    name: "Alerting"
    weight: 11
title: "Alerting"
date: 2016-10-25T17:24:32-07:00
---

You can configure Cloudprober to send alerts on probe failures. Alerts
configuration consists of mainly two parts:

- Alert condition
- Notifier

## Alert Condition

Alert condition is defined in terms of number of failures (`failures`) out of a
number of attempts (`total`). For example, if alert condition is specified as:
`condition {failures: 3, total: 5}`, an alert will be triggered if 3 probes have
failed out of the last 5 attempts.

If no condition is specified, both `failures` and `total` are assumed to be 1,
i.e., alert will trigger on the first probe failure itself.

Alert condition definition lets you take care of both the cases: _continuous
failures_ and _sporadic failures_. For example, if you run a probe every 30s and
alert condition is `{failures: 4, total: 10}`, alert will trigger either after 2
minutes of consecutive failures, or if probe has failed 4 times in last 5
minutes.

More examples and explanation:

`F=failure, S=success`

- ` condition { failure: 3 }` or `condition { failures: 3, total: 3} `

  Trigger an alert on 3 consecutive failures. A pattern like **`F F F`** will
  immediately trigger an alert, but **`S S F F S S F F S S`** will not, even
  though failures are happening quite regularly. You could catch the second
  pattern by configuring the alert condition as `{failures: 3, total: 5}`.

- `condition { failure: 3, total: 6 }`: Alert will trigger on 3 failures in the
  last 6 attempts. A pattern like **`F F F`** will immediately trigger an alert,
  so if probe interval is 10s, you'll get an alert within 20-30s of incident
  starting. A pattern like **`F S F S S F`** or **`F F S S S F`** will also
  trigger an alert.

## Notifier Configuration

Cloudprober currently supports the following notification targets:

- Email
- PagerDuty
- Slack
- Arbitrary Command
