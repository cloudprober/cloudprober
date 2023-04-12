---
menu:
  docs:
    parent: 'how-to'
    weight: 23
title: 'Validators'
date: 2022-10-01T17:24:32-07:00
---

Validators allow you to run checks on the probe request output (if any). For
example, you can specify if you expect the probe output to match a certain
regex or return a certain status code (for HTTP). You can configure more than
one validators and all validators should succeed for the probe to be marked as
success.

```shell
probe {
  name: "google_homepage"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }

  interval_msec: 10000 # Probe every 10s

  # This validator should succeed.
  validator {
      name: "status_code_2xx"
      http_validator {
          success_status_codes: "200-299"
      }
  }

  # This validator will fail, notice missing 'o' in our regex.
  validator {
      name: "gogle_re"
      regex: "gogle"
  }
}
```

(Full listing: https://github.com/cloudprober/cloudprober/blob/master/examples/validators/cloudprober_validator.cfg)

To make the debugging easier, validation failures are logged and exported as an
independent map counter -- _validation_failure_, with _validator_ key. For
example, the above example will result in the following counters being
exported after 5 runs:

```shell
total{probe="google_homepage",dst="www.google.com"} 5
success{probe="google_homepage",dst="www.google.com"} 0
validation_failure{validator="status_code_2xx",probe="google_homepage",dst="www.google.com"} 0
validation_failure{validator="gogle_re",probe="google_homepage",dst="www.google.com"} 5
```

Note that validator counter will **not** go up if probe fails for other
reasons, for example web server timing out. That's why you typically don't want
to alert only on validation failures. That said, in some cases, validation
failures could be the only thing you're interested in, for example, if you're
trying to make sure that a certain copyright is always present in your web
pages or you want to catch data integrity issues in your network.

Let's take a look at the types of validators you can configure.

## Regex Validator

Regex validator simply checks for a regex in the probe request output. It works
for all probe types except for UDP and UDP_LISTENER - these probe types don't
support any validators at the moment.

## HTTP Validator

HTTP response validator works only for the HTTP probe type. You can currently
use HTTP validator to define success and failure status codes (represented by
success_status_codes and failure_stauts_codes in the config):

- If _failure_status_codes_ is defined and response status code falls within
  that range, validator is considered to have failed.
- If _success_status_codes_ is defined and response status code _does not_
  fall within that range, validator is considered to have failed.
- If _failure_header_ is defined and HTTP response include specified header and
  there are matching values, validator is considered to have failed. Leaving
  _value_regex_ empty checks only for header name.
- If _success_header_ is defined and HTTP response _does not_ include specified header
  with matching values, validator is considered to have failed. Leaving
  _value_regex_ empty checks only for header name.

## Data Integrity Validator

Data integrity validator is designed to catch the packet corruption issues in
the network. We have a basic check that verifies that the probe output is made
up purely of a pattern repeated many times over.
