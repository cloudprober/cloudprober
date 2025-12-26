# Consul Surfacer Examples

This directory contains examples for registering Cloudprober instances with Consul.

## Overview

The Consul surfacer registers your Cloudprober instance as a service in Consul's service catalog. This makes your monitoring infrastructure discoverable and allows other services to:

- Find Cloudprober instances
- Check their health status
- Access their metrics endpoints
- Participate in service mesh (Consul Connect)

## Files

- `consul.cfg` - Complete example showing Consul service registration

## Quick Start

The simplest configuration to register with Consul:

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"
    service {
      name: "cloudprober"
      tags: "monitoring"
    }
    health_check {
      http_endpoint: "/status"
      interval: "10s"
    }
  }
}
```

## How It Works

1. **Registration**: When Cloudprober starts, the surfacer registers it as a service in Consul
2. **Health Checks**: A dedicated HTTP server answers Consul health check requests
3. **Discovery**: Other services can discover Cloudprober via Consul's DNS or API
4. **Deregistration**: On shutdown, the service is cleanly removed from Consul

## Configuration

### Basic Service Registration

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"

    service {
      name: "cloudprober"
      tags: "monitoring"
      tags: "production"
      port: 9313
    }

    health_check {
      http_endpoint: "/status"
      interval: "10s"
      timeout: "5s"
    }
  }
}
```

### With Custom Metadata

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"

    service {
      name: "cloudprober"
      tags: "monitoring"
    }

    metadata {
      key: "environment"
      value: "production"
    }
    metadata {
      key: "region"
      value: "us-west-2"
    }
  }
}
```

### With TLS

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "consul.example.com:8501"

    tls {
      ca_file: "/etc/cloudprober/ca.crt"
      cert_file: "/etc/cloudprober/client.crt"
      key_file: "/etc/cloudprober/client.key"
    }

    datacenter: "dc1"
    token: "your-consul-token"

    service {
      name: "cloudprober"
    }
  }
}
```

### With Kubernetes Service Discovery

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    # Discover Consul address via Kubernetes service
    kubernetes_service {
      namespace: "default"
      service_name: "consul"
      port: "8500"
    }

    service {
      name: "cloudprober"
    }
  }
}
```

### With Consul Connect (Service Mesh)

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"

    service {
      name: "cloudprober"
      enable_connect: true  # Enable Consul Connect
    }
  }
}
```

## Health Check Endpoint

The surfacer automatically starts an HTTP server for Consul health checks on port `servicePort + 1`:

- **Service port**: 9313 (default, where Prometheus metrics are served)
- **Health check port**: 9314 (automatically assigned)
- **Health endpoint**: `/status` (configurable)

The endpoint returns:
- `200 OK` when healthy
- `503 Service Unavailable` when unhealthy

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `address` | Consul server address | `localhost:8500` |
| `datacenter` | Consul datacenter | (default) |
| `token` | Consul ACL token | (none) |
| `service.name` | Service name | `cloudprober` |
| `service.id` | Service ID | auto-generated |
| `service.tags` | Service tags | (none) |
| `service.port` | Service port | 9313 |
| `service.address` | Service address | hostname |
| `service.enable_connect` | Enable Consul Connect | false |
| `health_check.http_endpoint` | Health check path | `/status` |
| `health_check.interval` | Check interval | `10s` |
| `health_check.timeout` | Check timeout | `5s` |
| `health_check.deregister_critical_service_after` | Deregister after | `1m` |
| `metadata` | Custom metadata | (none) |
| `deregister_on_shutdown` | Deregister on exit | true |

## Verification

After starting Cloudprober with the Consul surfacer:

```bash
# List all services
consul catalog services

# Get details about cloudprober service
consul catalog service cloudprober

# Check health status
consul health service cloudprober

# Query via DNS
dig @localhost -p 8600 cloudprober.service.consul
```

## Use Cases

1. **Service Discovery**: Other monitoring tools can discover Cloudprober instances
2. **Load Balancing**: Use Consul's service discovery for load balancing across multiple Cloudprober instances
3. **Health Monitoring**: Monitor Cloudprober's own health via Consul
4. **Service Mesh**: Integrate with Consul Connect for encrypted communication

## Prerequisites

1. Running Consul agent:
   ```bash
   consul agent -dev
   ```

2. Network access to Consul (default: `localhost:8500`)

3. Optional: Consul ACL token (if ACLs are enabled)

## Examples

See `consul.cfg` for a complete working example.

## See Also

- [Consul Targets Examples](../../targets/consul/) - Use Consul for target discovery
- [Consul Documentation](https://www.consul.io/docs)
- [Consul Service Registration](https://www.consul.io/docs/discovery/services)
