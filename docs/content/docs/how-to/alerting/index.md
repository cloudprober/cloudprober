---
menu:
  docs:
    parent: how-to
    name: "Alerting"
    weight: 11
title: "Alerting"
date: 2016-10-25T17:24:32-07:00
---

You can configure Cloudprober to send alerts on probe failures. Alerts are
configured per probe and each probe can have multiple alerts with independent
configuration. Alert
[configuration](/docs/config/probes/#cloudprober.probes.alerting.AlertConf)
consists of mainly two parts:

- [Alert condition](/docs/config/probes/#cloudprober.probes.alerting.Condition)
- [Notification config](/docs/config/probes/#cloudprober.probes.alerting.NotifyConfig)

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

## Notification Targets

Cloudprober currently supports the following notification targets:

- [Email](/docs/config/probes/#cloudprober.probes.alerting.Email)
- [PagerDuty](/docs/config/probes/#cloudprober.probes.alerting.PagerDuty)
- [Slack](/docs/config/probes/#cloudprober.probes.alerting.Slack)
- [Command](/docs/config/probes/#cloudprober.probes.alerting.NotifyConfig)

Configuration documentation has more details on each of them. You can configure
multiple notification targets for an alert.

## Notification Configuration

In addition to the alert condition and notification targets, you can also
configure the information included in your alert notification. Please look at
the alert config documentation for more details about the individual fields:
[AlertConf](/docs/config/probes/#cloudprober.probes.alerting.AlertConf).
