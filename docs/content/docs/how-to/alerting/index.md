---
menu:
  docs:
    parent: how-to
    name: "Alerting"
    weight: 10
title: "Alerting"
date: 2023-10-04T17:24:32-07:00
---

You can configure Cloudprober to send alerts on probe failures. Alerts are
configured per probe and each probe can have multiple alerts with independent
configuration. Alert
[configuration](/docs/config/alerting/#cloudprober_alerting_AlertConf) consists
of mainly two parts:

- [Alert condition](/docs/config/alerting/#cloudprober_alerting_Condition)
- [Notification config](/docs/config/alerting/#cloudprober_alerting_NotifyConfig)

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

- ` condition { failures: 3 }` or `condition { failures: 3, total: 3} `

  Trigger an alert on 3 consecutive failures. A pattern like **`F F F`** will
  immediately trigger an alert, but **`S S F F S S F F S S`** will not, even
  though failures are happening quite regularly. You could catch the second
  pattern by configuring the alert condition as `{failures: 3, total: 5}`.

- `condition { failures: 3, total: 6 }`: Alert will trigger on 3 failures in the
  last 6 attempts. A pattern like **`F F F`** will immediately trigger an alert,
  so if probe interval is 10s, you'll get an alert within 20-30s of incident
  starting. A pattern like **`F S F S S F`** or **`F F S S S F`** will also
  trigger an alert.

## Alerts Dashboard

Cloudprober comes with an _alerts dashboard_ that you can access at the
`/alerts` URL. Alerts dashboard shows currently firing and 20 historical alerts.

## Notifications

When you add alerts, you'd probably also want to be notified when they fire. You
can do that using the `notify` config block:

```yaml
# This example is in YAML format. You can use the original textpb format too.
# See https://cloudprober.org/docs/config/alerting/cloudprober_alerting_AlertConf
probe:
  ...
  alert:
    notify:
      pager_duty:
        routing_key: "..."
      slack:
        webhook_url: "..."
```

Above example configures two notifications: PagerDuty & Slack. Cloudprober
currently supports the following notification targets:

- [Email](/docs/config/alerting/#cloudprober_alerting_Email)
- [PagerDuty](/docs/config/alerting/#cloudprober_alerting_PagerDuty)
- [Opsgenie](/docs/config/alerting/#cloudprober_alerting_Opsgenie)
- [Slack](/docs/config/alerting/#cloudprober_alerting_Slack)
- [Command](/docs/config/alerting/#cloudprober_alerting_NotifyConfig)
- [HTTP](/docs/config/alerting/#cloudprober_alerting_NotifyConfig)

Configuration documentation (linked above) has more details on each of them.

## Notification Fields

You can customize the information included in the alert notification. The
following table shows the top-level notification parameters, corresponding
config fields, and their default values.

| Info          | Configuration Field      | Default                                                                                                                                                                                                               |
| ------------- | ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Dashboard URL | `dashboard_url_template` | http://localhost:9313/status?probe=@probe@                                                                                                                                                                            |
| Playbook URL  | `playbook_url_template`  | ""                                                                                                                                                                                                                    |
| Summary       | `summary_template`       | Cloudprober alert "`@alert@`" for "`@target@`"                                                                                                                                                                        |
| Details       | `details_template`       | Cloudprober alert "`@alert@`" for "`@target@`":<br/>Failures: `@failures@` out of `@total@` probes<br/>Failing since: `@since@`<br>Probe: `@probe@`<br/> Dashboard: `@dashboard_url@`<br/> Playbook: `@playbook_url@` |

As you see here, you can embed alert information in notifications using
placeholders like `@field@`. For example if you host your alert playbooks at
https://playbook/<alertname>, you can configure `playbook_url` to be
`https://playbook/@alert@`. Cloudprober supports the following alert fields
placeholders:

| Placeholder                                                   | Alert field                                                                                             |
| ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `@alert@`                                                     | Alert name. Same as probe name if alert name is not configured                                          |
| `@probe@`                                                     | Probe name                                                                                              |
| `@target@`                                                    | Target name                                                                                             |
| `@target.label.<label>@`                                      | Target label value, e.g. if target has a label: env=prod, @target.label.env@ will be replaced with prod |
| `@since@`                                                     | Alert start time                                                                                        |
| `@failure@`                                                   | Failure count that caused the alert                                                                     |
| `@total@`                                                     | Total number of probes                                                                                  |
| `@target_ip@`                                                 | Target IP if available. It only works for targets discovered by Cloudprober                             |
| `@dashboard_url@`, `@playbook_url@`, `@summary@`, `@details@` | See the table above.                                                                                    |
