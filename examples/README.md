# Cloudprober Examples

This directory contains various examples demonstrating different features and use cases of Cloudprober. Below is a table that maps features to their corresponding subdirectories.

| Feature | Description | Subdirectory |
|---------|-------------|--------------|
| Additional Labels | Examples of adding custom labels to metrics | `additional_label/` |
| AWS Discovery | Discovering and probing AWS resources (EC2, RDS) | `targets/aws/` |
| Extensions | How to extend Cloudprober with custom probes and targets | `extensions/` |
| External Probes | Examples of external probes in different languages | `external/` |
| File-based Targets | Configuring targets using files | `file_based_targets/` |
| gRPC | Examples of gRPC probes and servers | `grpc/` |
| Include Files | Splitting configuration into multiple files | `include/` |
| OAuth | Authentication examples using OAuth | `oauth/` |
| Scheduling | Run probes at specific times of the day | `schedule/` |
| Surfacers | Different ways to export metrics | `surfacers/` |
| Targets | Various target configurations | `targets/` |
| Templates | Using Go templates in configurations | `templates/` |
| Validators | Examples of response validators | `validators/` |

## Getting Started

Each subdirectory contains example configurations and, where applicable, additional documentation. Refer to the specific subdirectory for detailed information about each feature.

## Contributing

If you'd like to contribute additional examples, please follow the existing directory structure, include a README.md file explaining the example, and update the table above.
