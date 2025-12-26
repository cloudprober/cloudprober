# Consul Targets Examples

This directory contains examples for using Consul to discover targets for probes.

## Files

- `consul.cfg` - Complete example showing Consul target discovery with the terser syntax

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
