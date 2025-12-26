# Consul Surfacer

This package implements a Consul surfacer that registers the Cloudprober instance with Consul's service catalog.

## Features

- **Service Registration**: Register Cloudprober as a Consul service
- **Health Checks**: Automatic health check registration with HTTP endpoint
- **Metadata**: Publish custom metadata about the instance
- **Service Tags**: Tag the service for easy discovery
- **Consul Connect**: Optional service mesh integration
- **Auto-deregistration**: Clean deregistration on shutdown
- **TLS Support**: Secure connections to Consul
- **Kubernetes Integration**: Discover Consul address via Kubernetes service

## Generating Proto Files

After making changes to the proto files, regenerate the Go code using the centralized script:

```bash
# From the project root
./tools/gen_pb_go.sh
```

This script automatically finds and regenerates all proto files in the project, including the Consul surfacer protos.

## Configuration

See the [examples directory](../../../examples/consul/) for complete configuration examples.

### Basic Configuration

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

### With Custom Metadata

```textproto
surfacer {
  type: CONSUL
  consul_surfacer {
    address: "localhost:8500"
    service {
      name: "cloudprober"
      tags: "monitoring"
      tags: "production"
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

## How It Works

1. **Initialization**: When Cloudprober starts, the surfacer:
   - Creates a Consul client
   - Determines service details (name, ID, address, port)
   - Starts a health check HTTP server
   - Registers the service with Consul

2. **Health Monitoring**: The surfacer:
   - Runs an HTTP server for Consul health checks
   - Responds to health check requests
   - Can update health status based on probe failures

3. **Shutdown**: When Cloudprober exits, the surfacer:
   - Deregisters the service from Consul (if configured)
   - Shuts down the health check server

## Health Check Endpoint

The surfacer starts a dedicated HTTP server for Consul health checks on `servicePort + 1`.

**Default:**
- Service port: 9313
- Health check port: 9314
- Health check endpoint: `/status`

The endpoint returns:
- `200 OK` when healthy
- `503 Service Unavailable` when unhealthy

## Service Discovery

Once registered, other services can discover Cloudprober via Consul:

```bash
# Query Cloudprober services
consul catalog service cloudprober

# Get health status
consul health service cloudprober
```

## Use Cases

1. **Service Mesh Integration**: Register Cloudprober in Consul Connect for encrypted service-to-service communication

2. **Dynamic Discovery**: Allow other monitoring tools to discover Cloudprober instances

3. **Health Monitoring**: Monitor Cloudprober's health via Consul

4. **Load Balancing**: Use Consul's service discovery for load balancing across multiple Cloudprober instances

## Development

### Running Tests

```bash
go test ./internal/surfacers/consul/...
```

### Code Structure

- `consul.go`: Main surfacer implementation
- `proto/config.proto`: Configuration proto definition

### Health Status Control

The surfacer exposes a `SetHealthStatus(bool)` method to programmatically control health status:

```go
surfacer.SetHealthStatus(false) // Mark as unhealthy
surfacer.SetHealthStatus(true)  // Mark as healthy
```

This can be extended to automatically mark the instance as unhealthy based on probe failures.
