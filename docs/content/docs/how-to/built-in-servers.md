---
menu:
  docs:
    parent: "how-to"
    weight: 32
title: "Built-in Servers"
---

Cloudprober has a few built in servers. This is useful when you are probing that
a connection is working, or as a baseline to compare the probing results from
your actual service to.

## HTTP

```shell
server {
  type: HTTP
  http_server {
    port: 8080
  }
}
```

This creates an HTTP server that responds on port `8080`. By default it will
respond to the following endpoints:

- `/healthcheck`
- `/lameduck`

```shell
server {
  type: HTTP
  http_server {
    port: 8080
    pattern_data_handler {
      response_size: 1024
    }

    pattern_data_handler {
      response_size: 4
      pattern: "four"
    }
  }
}
```

This adds two endpoints to the HTTP server:

- `/data_1024` which responds with 1024 bytes of
  `cloudprobercloudprobercloudprober`.
- `/data_4` which responds with `four`.

See [ServerConf](/docs/config/servers/#cloudprober_servers_http_ServerConf) for
all HTTP server configuration options.

## UDP

A Cloudprober UDP server can be configured to either echo or discard packets it
receives.

```shell
server {
  type: UDP
  udp_server {
    port: 85
    type: ECHO
  }
}

server {
  type: UDP
  udp_server {
    port: 90
    type: DISCARD
  }
}
```

See [ServerConf](/docs/config/servers/#cloudprober_servers_udp_ServerConf) for
all UDP server configuration options.

## GRPC

See [ServerConf](/docs/config/servers/#cloudprober_servers_grpc_ServerConf) for
all GRPC server configuration options.
