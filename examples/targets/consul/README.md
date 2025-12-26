# Consul Targets Examples

This directory contains examples for using Consul to discover targets for probes.

## Files

- `consul.cfg` - Complete example showing Consul target discovery with the terser syntax
- `global_consul.cfg` - Example showing how to use global Consul configuration across multiple probes

## Environment Variables

The Consul targets provider respects standard Consul environment variables. This allows you to configure Consul connection settings without hardcoding them in your configuration files.

**Supported environment variables:**
- `CONSUL_HTTP_ADDR`: Consul server address (e.g., `localhost:8500`)
- `CONSUL_HTTP_TOKEN`: ACL token for authentication
- `CONSUL_HTTP_TOKEN_FILE`: Path to file containing ACL token
- `CONSUL_HTTP_SSL`: Enable HTTPS (set to `true`)
- `CONSUL_HTTP_SSL_VERIFY`: Verify SSL certificates (default: `true`)
- `CONSUL_CACERT`: Path to CA certificate
- `CONSUL_CAPATH`: Path to directory of CA certificates
- `CONSUL_CLIENT_CERT`: Path to client certificate
- `CONSUL_CLIENT_KEY`: Path to client key
- `CONSUL_DATACENTER`: Datacenter name
- `CONSUL_NAMESPACE`: Namespace (Consul Enterprise only)
- `CONSUL_TLS_SERVER_NAME`: TLS server name for SNI
- `CONSUL_HTTP_AUTH`: HTTP basic auth (format: `username:password`)

**Configuration precedence:**
1. Values explicitly set in configuration files take highest precedence
2. Environment variables are used if no config value is set
3. Default values are used if neither config nor environment variables are set

**Example using environment variables:**
```bash
export CONSUL_HTTP_ADDR="consul.example.com:8500"
export CONSUL_HTTP_TOKEN="your-acl-token"
export CONSUL_DATACENTER="dc1"

# Now you can omit these from your config:
cloudprober --config_file=cloudprober.cfg
```

With environment variables set, your config becomes simpler:
```textproto
probe {
  name: "web_services"
  type: HTTP
  targets {
    consul {
      # Uses CONSUL_HTTP_ADDR from environment
      services: "web-.*"
      health_status: "passing"
    }
  }
  http_probe { relative_url: "/health" }
}
```

## Quick Start

The simplest way to use Consul for target discovery:

```textproto
probe {
  name: "web_services"
  type: HTTP
  targets {
    consul {
      address: "localhost:8500"
      services: "web-.*"
      health_status: "passing"
    }
  }
  http_probe {
    relative_url: "/health"
  }
}
```

## Global Consul Configuration

You can configure Consul connection settings globally to avoid repeating them in each probe:

```textproto
global_targets_options {
  global_consul_options {
    address: "localhost:8500"
    datacenter: "dc1"

    # Auto-discover Consul in Kubernetes
    kubernetes_service {
      namespace: "default"
      service_name: "consul"
      port: "8500"
    }
  }
}

probe {
  name: "web_services"
  type: HTTP
  targets {
    consul {
      # Inherits address from global_consul_options
      services: "web-.*"
    }
  }
  http_probe { relative_url: "/health" }
}
```

**Benefits:**
- Configure Consul address, datacenter, and TLS settings once
- Share configuration across multiple probes
- Local probe configuration can override global settings
- Supports Kubernetes service discovery for auto-detecting Consul

**Note:** Global configuration is currently only available for targets. Surfacers must configure Consul connection settings directly.

See `global_consul.cfg` for a complete example.

## Syntax

Cloudprober supports two syntaxes for Consul targets:

### 1. Direct Consul Targets (Recommended - Terser)

```textproto
targets {
  consul {
    address: "localhost:8500"
    services: "web-.*"  # Regex pattern for service names
    tags: "http"        # Filter by tags
    health_status: "passing"
  }
}
```

### 2. RDS Targets (Verbose - for advanced use cases)

```textproto
# First, configure RDS server (global config)
rds_server {
  provider {
    id: "consul"
    consul_config {
      address: "localhost:8500"
      services {
        tag_filter: "http"
        health_status: "passing"
      }
    }
  }
}

# Then use in probe
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

The RDS syntax is more verbose but allows for:
- Running a dedicated RDS server that multiple cloudprober instances connect to
- More complex filtering scenarios
- Sharing Consul configuration across multiple probes

## Resource Types

### Services (Default)

Discover Consul services:

```textproto
consul {
  address: "localhost:8500"
  services: "web-.*"  # Service name pattern (regex)
  tags: "http"        # All specified tags must match
  health_status: "passing"
}
```

### Health Checks

Discover health checks:

```textproto
consul {
  address: "localhost:8500"
  health_checks: "api"  # Health checks for "api" service
}
```

### Nodes

Discover Consul nodes:

```textproto
consul {
  address: "localhost:8500"
  nodes: ""  # All nodes
}
```

## Filtering

You can apply additional filters using the RDS filter syntax:

```textproto
consul {
  address: "localhost:8500"
  services: ""

  filter {
    key: "node"
    value: "prod-.*"  # Only targets on nodes matching pattern
  }

  filter {
    key: "labels.service"
    value: "frontend"
  }
}
```

### Available Filter Keys for Services:
- `name` - Service name (regex)
- `node` - Node name (regex)
- `tag` - Service tag (matches any tag with regex)
- `address` - Service address (regex)
- `labels.*` - Any label

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `address` | Consul server address | `localhost:8500` |
| `datacenter` | Consul datacenter | (default datacenter) |
| `token` | Consul ACL token | (none) |
| `services` | Service name pattern | - |
| `health_checks` | Health check filter | - |
| `nodes` | Node filter | - |
| `tags` | Service tags (all must match) | (none) |
| `health_status` | Health statuses to include | (none) |
| `re_eval_sec` | Target refresh interval | 30 |

## Prerequisites

1. Running Consul server:
   ```bash
   consul agent -dev
   ```

2. Network access to Consul (default: `localhost:8500`)

3. Optional: Consul ACL token (if ACLs are enabled)

## Examples

See `consul.cfg` for complete working examples.

## See Also

- [Consul Surfacer Examples](../../surfacers/consul/) - Register Cloudprober with Consul
- [Consul Documentation](https://www.consul.io/docs)
