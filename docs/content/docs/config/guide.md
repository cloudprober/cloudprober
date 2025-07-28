---
menu:
  docs:
    parent: "config"
    weight: 01
title: "Config Guide"
---

Cloudprober offers flexible and powerful options to manage the configs. This document details the key features of Cloudprober's configuration system.

## Config Language

Cloudprober uses Protocol Buffers (protobufs) as the foundation for its configuration schema. This allows configurations to be written in multiple formats, providing flexibility for different use cases. The supported formats are:

- **Textproto**: A human-readable text format for protobufs, directly aligned with the protobuf schema defined in `config.proto`. It is straightforward and closely mirrors the underlying structure. Example: [cloudprober.cfg](https://github.com/cloudprober/cloudprober/blob/main/config/testdata/unmarshal_test/cloudprober.cfg)
- **YAML**: A popular, human-readable format that can be used to define Cloudprober configurations. YAML configs are converted to protobufs internally. Example: [cloudprober.yaml](https://github.com/cloudprober/cloudprober/blob/main/config/testdata/unmarshal_test/cloudprober.yaml)
- **Jsonnet**: A data templating language that allows for more programmatic configuration. Jsonnet configs are also converted to protobufs for processing. Example: [cloudprober.jsonnet](https://github.com/cloudprober/cloudprober/blob/main/config/testdata/unmarshal_test/cloudprober.jsonnet)

For example, a simple Cloudprober configuration in YAML might look like this:

```yaml
probe:
  - name: "http_probe"
    type: HTTP
    targets:
      host_names: "www.example.com"
    interval_msec: 5000
```

The same configuration in textproto would be:

```protobuf
probe {
  name: "http_probe"
  type: HTTP
  targets {
    host_names: "www.example.com"
  }
  interval_msec: 5000
}
```

All formats are interchangeable, as they are parsed into the same protobuf structure. Choose the format that best suits your workflow or tooling preferences.

## Building Blocks

At its core, a Cloudprober configuration consists of a collection of distinct configuration entries, each serving a specific purpose. Here are the most common entry types:

- **[Probes](/docs/overview/probe)**: Probes are Cloudprober's primary building blocks. Most of the time you'll be working with these. Each probe entry defines a probe that Cloudprober will execute to monitor your systems. All probes have some common fields like *type*, *name*, *targets*, *interval*, *alert*, etc and some probe-type specific fields, e.g. HTTP probes have fields like *url*, *method*, etc. See [probes config reference](/docs/config/latest/probes) for more details on actual fields.
- **[Surfacers](/docs/surfacers/overview)**: Surfacers are responsible for exporting probe results to various monitoring systems (e.g., Prometheus, Stackdriver). It's uncommon to have more than one surfacer, and in most cases specifying no explicit surfacer, hence relying on the default surfacer works very well.
- **[Servers]({{<ref "built-in-servers">}})**: These are very rare. You add servers if you want Cloudprober to act as targets (yes, you can do that). This is typically used to monitor network and load balancers.

## Programmability

Cloudprober configurations support programmability to enable dynamic and reusable configs. This is achieved through two primary mechanisms:

### Go Text Templates

Cloudprober integrates [Go text templates](https://pkg.go.dev/text/template) to add programming capabilities like loops and conditionals. These templates allow you to dynamically generate configurations based on variables or logic. For example, you can use loops to create multiple probes with similar configurations or conditionals to include specific settings based on environment variables.

Example of a Go text template with a loop:

```protobuf
{{ range $host := (list "cloudprober.org" "pagenotes.manugarg.com") }}
probe {
  name: "probe_{{ $host }}"
  type: HTTP
  targets {
    host_names: "{{ $host }}"
  }
  interval_msec: 5000
}
{{ end }}
```

### Jsonnet-Based Programmability

When using Jsonnet as the configuration format, you gain access to Jsonnet's native programmability features, such as functions, variables, and conditionals. This is particularly useful for complex configurations that require reusable logic or abstraction. Jsonnet's features are automatically available when you write your config in Jsonnet, eliminating the need for additional templating.

Example of a Jsonnet config:

```jsonnet
local probeTemplate(name, host) = {
  probe: [{
    name: name,
    type: 'HTTP',
    targets: {
      host_names: host,
    },
    interval_msec: 5000,
  }],
};

probeTemplate('http_probe', 'www.example.com')
```

Jsonnet's flexibility makes it ideal for generating configurations programmatically, especially in large-scale deployments.[](https://github.com/cloudprober/cloudprober/blob/master/config/proto/config.proto)[](https://news.ycombinator.com/item?id=35325488)

## Useful Inbuilt Functions and Variables

Cloudprober enhances Go text templates with additional functions and variables, including those provided by the [Sprig library](https://masterminds.github.io/sprig/) and custom Cloudprober-specific extensions.

### Sprig Library Functions

Cloudprober incorporates the [Sprig library](https://masterminds.github.io/sprig/), which provides over 100 functions for string manipulation, lists, math, date handling, and more. These functions can be used within Go text templates to add advanced logic to your configurations. Examples include:

- `toUpper`: Converts a string to uppercase.
- `randAlphaNum`: Generates a random alphanumeric string.
- `date`: Formats a date or time.

Example using Sprig functions:

```protobuf
{{ $host := "web1" }}
probe {
  name: "{{ $host | toUpper }}"
  type: HTTP
  targets {
    host_names: "{{ $host }}"
  }
  interval_msec: {{ 5000 + (randInt 1 1000) }}
}
```

In this example, `toUpper` capitalizes the probe name, and `randInt` adds a random interval offset.

### Cloudprober-Specific Extensions

Cloudprober provides custom template extensions, defined in [config_tmpl.go](https://github.com/cloudprober/cloudprober/blob/main/config/config_tmpl.go), to support specific use cases. A key extension is `configDir` variable, which retrieves the directory of the current configuration file. This is useful for referencing additional files relative to the config file's location.

Example using `configDir`:

```protobuf
probe {
  name: "file_based_probe"
  type: HTTP
  targets {
    file_targets {
      file_path: "{{ configDir }}/targets.textpb"
      re_eval_sec: 300
    }
  }
}
```

Here, `configDir` resolves to the directory of the config file, allowing you to reference a `targets.textpb` file in the same directory. Other Cloudprober-specific functions include utilities to access GCE Custom Metadata and declare secret environment variables that don't show up in the web interface.

## Splitting Config into Multiple Files

Cloudprober supports splitting configurations across multiple files using the `include` directive, which is particularly useful for modularizing large configurations. This allows you to organize probes, targets, or other settings into separate files for better maintainability.

### Using the `include` Directive

The `include` directive can be used to import content from other files. For example, you can define common probes in separate files and include them in your main configuration.

NOTE: `include` doesn't work very well with yaml files.

Example directory structure:

```
cloudprober/
├── cloudprober.cfg
├── probes/
│   ├── http_probe.cfg
│   ├── dns_probe.cfg
```

Main configuration (`cloudprober.cfg`):

```proto
include "probes/http_probe.cfg"
include "probes/dns_probe.cfg"
```

Example `probes/http_probe.cfg`:

```proto
probe {
  name: "http_probe"
  type: HTTP
  ...
}
```

The `include` directive reads the content of `http_probe.cfg` and inserts it into the main config. This approach allows you to reuse probe definitions across multiple configurations or environments.

For a practical example, see the Cloudprober repository's **[examples/include](https://github.com/cloudprober/cloudprober/tree/main/examples/include)** directory, which demonstrates how to structure and use multiple configuration files.

## Accessing Configuration via Webserver

Cloudprober provides a webserver interface to access the current configuration in different forms, which is useful for debugging. The following endpoints are available on the Cloudprober webserver (typically accessible at `http://<cloudprober-host>:9313`):

- **`/config-running`**: Displays the parsed and running configuration, reflecting the final state of the configuration after all templates (e.g., Go text templates or Jsonnet) and `include` directives have been processed and config has been converted to protobufs. This is the actual configuration being used by Cloudprober at runtime.
  
- **`/config-parsed`**: Shows the parsed configuration after processing templates and includes, but before converting to protobufs. This will be more readable and familiar than what you get from `/config-running`.

- **`/config`**: Returns the raw, unprocessed configuration as provided to Cloudprober (e.g., the original textproto, YAML, or Jsonnet file). This is useful for verifying that cloudprober is reading the correct config file.

Example usage:
```bash
curl http://localhost:9313/config-parsed
```

## Conclusion

Cloudprober's configuration system, built on protobufs, offers flexibility through multiple formats (textproto, YAML, Jsonnet) and powerful programmability via Go text templates and Jsonnet. The integration of Sprig and Cloudprober-specific extensions, like `configDir`, enhances dynamic configuration capabilities. Splitting configs into multiple files using the `include` directive further improves modularity and maintainability. For more examples, refer to the [Cloudprober documentation](https://cloudprober.org) and the [examples/include](https://github.com/cloudprober/cloudprober/tree/main/examples/include) directory on GitHub.