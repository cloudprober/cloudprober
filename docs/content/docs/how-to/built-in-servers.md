---
menu:
  docs:
    parent: "how-to"
    weight: 32
title: "Built-in Servers"
---

Cloudprober comes with some custom servers that can be enabled through
configuration. These servers can act as targets for the other probes -- for
example, you can run two Cloudprober instances on two different machines and
have one instance's servers act as targets and other instance probe those
targets.

These servers can come in handy when the goal is to monitor the underlying
infrastructure: e.g. network or load balancers.

<pre>
Cloudprober (probes) ===(Network)===> Cloudprober (servers)
</pre>

## HTTP Server

```shell
server {
  type: HTTP
  http_server {
    port: 8080
  }
}
```

This creates an HTTP server that responds on the port `8080`. This HTTP server
supports the following two endpoints by default:

- `/healthcheck` - returns 'OK' if instance is not in the lameduck mode.
- `/lameduck` - returns the lameduck status (true/false).

Lameduck mode is a mode in which a server is still running but is signaling that
it is about to go down for maintenance so new requests should not be sent to it.
This is typically used with load balancers to take out a backend for maintenance
without returning any actual errors.

TODO(manugarg): Document how a Cloudprober can be put in the lameduck mode.

### Data Handlers

You can also add custom data handlers to the above HTTP server:

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

Above configuration adds the following two URLs to the HTTP server:

- `/data_1024` which responds with 1024 bytes of
  `cloudprobercloudprober...(repeated)`.
- `/data_4` which responds with `four`.

These endpoints are useful to monitor other aspects of the underlying network
like MTU, and consistency (make sure data is not getting corrupted), etc.

See [this](/docs/config/servers/#cloudprober_servers_http_ServerConf) for all
HTTP server configuration options.

## UDP

UDP server can either echo packets back or completely ignore them. In echo mode,
you can use it along with the UDP probe type.

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
