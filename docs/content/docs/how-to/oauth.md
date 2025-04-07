---
menu:
  docs:
    parent: 'how-to'
    weight: 24
title: 'OAuth Based Authentication'
author: 'Manu Garg'
date: 2025-03-31T16:46:15-07:00
---

OAuth is the most ubiquitous authentication method used by services today. 
Cloudprober supports OAuth based authentication for HTTP and gRPC probes right
out of the box. You can add OAuth based authentication to your probes by
adding `oauth_config` stanza to your probe configuration, like this:

```bash
probe {
  name: "probe_with_oauth"
  type: HTTP
  http_probe {
    oauth_config: {
      # Add your oauth config here
    }
  }
}
```

## OAuth Configuration

Cloudprober allows you to get OAuth token from multiple sources. You can get it
from either a file, an HTTP request, an arbitrary command, k8s token file, or
GCE metadata.


```bash
# From file, say for example maintained and refreshed by another process
oauth_config: {
  file: "/path/to/bearer/token"
  refresh_interval_sec: 60 # Refresh token every 60 seconds
}

# From k8s token file
oauth_config: {
  k8s_local_token: true
}

# Run a command to generate the token
oauth_config: {
  # Token generator could do custom stuff like generate a short-lived token
  # from a private public key-pair (e.g. for Snowflake API).
  cmd: "{{configDir}}/scripts/token_generator.sh"
}
```

### Token Refresh Behavior

If you specify `refresh_interval_sec`, Cloudprober will refresh the token from
the same source at the specified interval. Otherwise, Cloudprober determines
the refresh mechanism based on token's expiry. If the token has an expiry
Cloudprober will simply refresh based on that (most common scenario), otherwise
Cloudprober will refresh the token every 30 seconds by default.

### HTTP Request

You can also retrieve the token from an HTTP based source.

```bash
oauth_config: {
  http_request: {
    token_url: "https://oauth2.googleapis.com/token"
    method: POST
    data: [
      "client_id=your-client-id",
      "client_secret=your-client-secret",
      "grant_type=client_credentials",
      "scope=your-scope"
    ]
  }
}
```

### Google OAuth

If you're in the Google ecosystem, running on Cloud Run or GKE for example, you
can simply specify the `google_credentials` stanza to retrieve token from multiple
sources.

```bash
# Use default credentials while running on GCP (GKE, Cloud Run, GCE, etc)
# This gets token from Application Default Credentials (ADC) if available or
# GCE metadata service
oauth_config: {
  google_credentials: {}
}

# Use JSON file
oauth_config: {
  google_credentials: {
    json_file: "/path/to/your/credentials.json"
  }
}
```

## Config Reference

See the [OAuth Config reference](/docs/config/main/oauth/#cloudprober_oauth_Config)
for all available options.
