# Consul RDS Provider

This package implements a Consul-based Resource Discovery Service (RDS) provider for Cloudprober.

## Features

- **Service Discovery**: Discover services registered in Consul
- **Health Checks**: Query Consul health checks
- **Nodes**: Discover Consul nodes
- **Filtering**: Filter by tags, names, health status, and metadata
- **Multiple Datacenters**: Support for Consul datacenters
- **TLS Support**: Secure connections to Consul
- **Kubernetes Integration**: Discover Consul address via Kubernetes service

## Generating Proto Files

After making changes to the proto files, regenerate the Go code using the centralized script:

```bash
# From the project root
./tools/gen_pb_go.sh
```

This script automatically finds and regenerates all proto files in the project, including the Consul RDS provider protos.

## Configuration

See the [examples directory](../../../examples/consul/) for complete configuration examples.

### Basic Configuration

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
```

### Using in Probes

```textproto
probe {
  name: "consul_services"
  type: HTTP
  targets {
    rds_targets {
      resource_path: "consul://services"
      filter {
        key: "name"
        value: "web-.*"
      }
    }
  }
}
```

## Resource Types

### Services (`consul://services`)

Discover services registered in Consul.

**Filters:**
- `name`: Service name (regex)
- `node`: Node name (regex)
- `tag`: Service tag (matches any tag)
- `address`: Service address (regex)
- `labels.*`: Label-based filtering

**Labels:**
- `service`: Service name
- `node`: Node name
- `health`: Health status (passing, warning, critical)
- `tag_*`: Service tags
- `meta_*`: Service metadata

### Health Checks (`consul://health_checks`)

Discover Consul health checks.

**Filters:**
- `name`: Check name (regex)
- `service`: Service name (regex)
- `node`: Node name (regex)
- `status`: Check status (regex)

### Nodes (`consul://nodes`)

Discover Consul nodes.

**Filters:**
- `name`: Node name (regex)
- `address`: Node address (regex)
- `datacenter`: Datacenter name (regex)

## Development

### Running Tests

```bash
go test ./internal/rds/consul/...
```

### Code Structure

- `consul.go`: Main provider implementation
- `services.go`: Services lister
- `health_checks.go`: Health checks lister
- `nodes.go`: Nodes lister
- `proto/config.proto`: Configuration proto definition
