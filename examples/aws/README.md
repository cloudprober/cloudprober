# AWS Discovery Examples

This directory contains examples for using Cloudprober's AWS resource discovery capabilities. These examples demonstrate how to automatically discover and probe AWS resources using the Resource Discovery Service (RDS).

## Prerequisites

Before running these examples, you need:

1. **AWS Credentials**: Configure AWS credentials using one of the following methods:
   - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
   - AWS credentials file (`~/.aws/credentials`)
   - IAM role (when running on EC2)
   - ECS task role (when running in ECS)

2. **IAM Permissions**: Your AWS credentials need permissions to describe the resources you want to discover. At minimum:
   - `ec2:DescribeInstances` for EC2 discovery
   - `rds:DescribeDBInstances` for RDS instance discovery
   - `rds:DescribeDBClusters` for RDS cluster discovery
   - `elasticache:DescribeCacheClusters` for ElastiCache cluster discovery
   - `elasticache:DescribeReplicationGroups` for ElastiCache replication group discovery

## Examples

### 1. EC2 Instances Discovery (`ec2_instances.cfg`)

A simple example showing how to discover and probe EC2 instances.

**Features:**
- Discovers all EC2 instances in us-east-1
- Probes instances using HTTP on port 80
- Automatically updates the target list every 5 minutes

**Run:**
```bash
cloudprober --config_file=ec2_instances.cfg
```

### 2. All AWS Resources (`rds_all_resources.cfg`)

A comprehensive example demonstrating discovery of multiple AWS resource types.

**Features:**
- Discovers EC2 instances, RDS instances/clusters, and ElastiCache clusters/replication groups
- Shows different probe types (HTTP and TCP)
- Demonstrates resource-specific configuration options

**Run:**
```bash
cloudprober --config_file=rds_all_resources.cfg
```

## Supported AWS Resource Types

| Resource Path | Description |
|--------------|-------------|
| `aws://ec2_instances` | EC2 instances |
| `aws://rds_instances` | RDS database instances |
| `aws://rds_clusters` | RDS database clusters (Aurora, etc.) |
| `aws://elasticache_clusters` | ElastiCache clusters |
| `aws://elasticache_replicationgroups` | ElastiCache replication groups |

## Configuration Options

### Common Options (all resource types)

- **region**: AWS region to query (required)
- **profile_name**: AWS profile name (optional)
- **re_eval_sec**: How often to refresh the resource list (default: 600 seconds)

### RDS-Specific Options (rds_instances, rds_clusters)

- **identifier**: Filter by specific DB identifier or ARN
- **filter**: AWS API filters (e.g., "db-instance-status=available")
- **include_shared**: Include resources shared from other AWS accounts

## Using Filters

You can filter discovered resources using RDS filters:

```protobuf
targets {
  rds_targets {
    resource_path: "aws://ec2_instances"

    # Filter by instance tags
    filter {
      key: "labels.Environment"
      value: "production"
    }
  }
}
```

## Remote RDS Server

For large deployments, you can run RDS server as a separate process to centralize AWS API calls:

**RDS Server instance:**
```protobuf
rds_server {
  provider {
    aws_config {
      region: "us-east-1"
      ec2_instances {}
    }
  }
}

grpc_port: 9314
```

**Cloudprober instance:**
```protobuf
probe {
  name: "ec2-probe"
  type: HTTP

  targets {
    rds_targets {
      rds_server_options {
        server_address: "rds-server:9314"
      }
      resource_path: "aws://ec2_instances"
    }
  }
}
```

## Additional Resources

- [RDS Documentation](https://cloudprober.org/docs/how-to/rds/)
- [AWS Provider Configuration](https://github.com/cloudprober/cloudprober/blob/main/internal/rds/aws/proto/config.proto)
- [Cloudprober Documentation](https://cloudprober.org/)
