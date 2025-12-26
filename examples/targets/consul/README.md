# Consul Integration Examples

This directory contains example configurations for using Cloudprober with HashiCorp Consul.

## Features

### 1. Consul RDS (Resource Discovery Service)

The Consul RDS provider allows Cloudprober to discover and probe services registered in Consul.

**Key Features:**
- Discover services, health checks, and nodes from Consul
- Filter by service tags, names, health status, and metadata
- Support for multiple Consul datacenters
- TLS support for secure connections
- Kubernetes service discovery for Consul address
- Automatic refresh of service lists

**Example:** See `consul_rds.cfg`

### 2. Consul Surfacer

The Consul surfacer registers the Cloudprober instance with Consul's service catalog, making it discoverable by other services.

**Key Features:**
- Register Cloudprober instance as a Consul service
- Automatic health check registration
- Custom service metadata and tags
- Consul Connect (service mesh) support
- Automatic deregistration on shutdown

**Example:** See `consul_surfacer.cfg`

## Prerequisites

1. **Consul Server**: You need a running Consul server or cluster
   ```bash
   # Start Consul in dev mode (for testing)
   consul agent -dev
   ```

2. **Network Access**: Ensure Cloudprober can reach the Consul server
   - Default: `localhost:8500`
   - Custom: Configure via `address` field

3. **Authentication** (if required):
   - Set Consul token in config: `token: "your-token"`
   - Or use environment variable: `CONSUL_HTTP_TOKEN`

## Configuration Examples

### Basic Service Discovery

```textproto
rds_server {
  provider {
    id: "consul"
    consul_config {
      address: "localhost:8500"
      services {
        tag_filter: "http"
        health_status: "passing"
      }
      re_eval_sec: 30
    }
  }
}

probe {
  name: "consul_http_services"
  type: HTTP
  targets {
    rds_targets {
      resource_path: "consul://services"
    }
  }
  http_probe {
    relative_url: "/health"
  }
}
```

### Register Cloudprober with Consul

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"
    service {
      name: "cloudprober"
      tags: "monitoring"
      port: 9313
    }
    health_check {
      http_endpoint: "/status"
      interval: "10s"
    }
  }
}
```

### Using Kubernetes Service Discovery

```textproto
consul_config {
  kubernetes_service {
    namespace: "default"
    service_name: "consul"
    port: "8500"
  }
  # ... rest of config
}
```

### TLS Configuration

```textproto
consul_config {
  address: "consul.example.com:8501"
  tls {
    ca_file: "/etc/cloudprober/ca.crt"
    cert_file: "/etc/cloudprober/client.crt"
    key_file: "/etc/cloudprober/client.key"
  }
}
```

## Resource Path Format

The Consul RDS provider supports these resource paths:

- `consul://services` - All services
- `consul://services/web` - Specific service named "web"
- `consul://health_checks` - All health checks
- `consul://health_checks/api` - Health checks for "api" service
- `consul://nodes` - All Consul nodes

## Current Limitation

Currently, Consul targets require using the `rds_targets {}` syntax. Unlike Kubernetes which has both `k8s {}` (terser) and RDS provider support, Consul only has the RDS provider.

**Current syntax** (what works now):
```textproto
targets {
  rds_targets {
    resource_path: "consul://services"
    filter {
      key: "labels.health"
      value: "passing"
    }
  }
}
```

**Desired terser syntax** (not yet implemented):
```textproto
targets {
  consul {
    address: "localhost:8500"
    services: "web-.*"
    tags: ["http"]
    health_status: "passing"
  }
}
```

A future enhancement would add the terser `consul {}` targets type to `targets/proto/targets.proto` for consistency with Kubernetes and GCE targets.

## Filtering

You can filter discovered resources using these filter keys:

### For Services:
- `name` - Service name (regex)
- `node` - Node name (regex)
- `tag` - Service tag (regex, matches any tag)
- `address` - Service address (regex)
- `labels.service` - Service label
- `labels.health` - Health status

### For Health Checks:
- `name` - Check name (regex)
- `service` - Service name (regex)
- `node` - Node name (regex)
- `status` - Check status (regex)

### For Nodes:
- `name` - Node name (regex)
- `address` - Node address (regex)
- `datacenter` - Datacenter name (regex)

## Running the Examples

1. **Start Consul**:
   ```bash
   consul agent -dev
   ```

2. **Register some test services**:
   ```bash
   # Register a web service
   curl -X PUT -d '{
     "Name": "web-frontend",
     "Tags": ["http", "production"],
     "Port": 8080,
     "Check": {
       "HTTP": "http://localhost:8080/health",
       "Interval": "10s"
     }
   }' http://localhost:8500/v1/agent/service/register
   ```

3. **Run Cloudprober with RDS**:
   ```bash
   cloudprober --config_file=examples/consul/consul_rds.cfg
   ```

4. **Run Cloudprober with Surfacer**:
   ```bash
   cloudprober --config_file=examples/consul/consul_surfacer.cfg
   ```

5. **Verify registration** (for surfacer):
   ```bash
   # Check registered services
   consul catalog services

   # Check Cloudprober service details
   consul catalog service cloudprober

   # Check health status
   consul health service cloudprober
   ```

## Advanced Use Cases

### Multi-Datacenter Setup

```textproto
consul_config {
  address: "consul.dc1.example.com:8500"
  datacenter: "dc1"
  services {
    # Will discover services in dc1
  }
}
```

### Combining RDS and Surfacer

You can use both the RDS provider and surfacer in the same configuration to:
1. Discover other services to probe
2. Register the Cloudprober instance itself

See the individual example files for complete configurations.

### Service Mesh Integration

```textproto
consul_surfacer {
  service {
    name: "cloudprober"
    enable_connect: true  # Enable Consul Connect
  }
}
```

## Troubleshooting

1. **Connection Issues**:
   - Verify Consul is running: `consul members`
   - Check network connectivity: `curl http://localhost:8500/v1/status/leader`
   - Review Cloudprober logs for error messages

2. **No Services Discovered**:
   - Check filters aren't too restrictive
   - Verify services are registered in Consul: `consul catalog services`
   - Check datacenter configuration matches

3. **Health Check Failures**:
   - Ensure health check endpoint is accessible
   - Verify port configuration is correct
   - Check Cloudprober is listening on the configured port

## References

- [Consul Documentation](https://www.consul.io/docs)
- [Cloudprober Documentation](https://cloudprober.org)
- [Consul Service Discovery](https://www.consul.io/docs/discovery/services)
- [Consul Health Checks](https://www.consul.io/docs/discovery/checks)
