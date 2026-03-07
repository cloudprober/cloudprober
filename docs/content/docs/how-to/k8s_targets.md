---
menu:
  docs:
    parent: "how-to"
    weight: 16
title: "Kubernetes Targets"
date: 2022-11-01T17:24:32-07:00
---

Cloudprober supports [dynamic
discovery]({{< ref targets.md >}}#dynamically-discovered-targets) of Kubernetes
resources (e.g. pods, endpoints, ingresses, etc) through the targets type
[`k8s`](https://github.com/cloudprober/cloudprober/blob/ad73fe489ea3ac69e7b0f81a465671df9adc8321/targets/proto/targets.proto#L40).

For example, the following config adds an HTTP probe for the endpoints named
`cloudprober` (equivalent to running _kubectl get ep cloudprober_).

```bash
probe {
  name: "pod-to-endpoints"
  type: HTTP

  targets {
    # Equivalent to kubectl get ep cloudprober
    k8s {
      endpoints: "cloudprober"
    }
  }

  # Note that the following http_probe automatically uses target's discovered
  # port.
  http_probe {
    relative_url: "/status"
  }
}
```

### Supported Resource and Filters

Cloudprober supports discovery for the following k8s resources:

- **Services**
- **Endpoints**
- **Pods**
- **Ingresses**

#### Filters

You can filter k8s resources using the following options:

- `name`: (regex) Resource name filter. It can be a regex. Example:
  ```shell
  # Endpoints with names ending in "service"
  targets {
    k8s {
        endpoints: ".*-service"
    }
  }
  ```
- `namespace`: Namespace filter. Example:
  ```shell
  # Ingresses in "prod" namespace, ending in "lb"
  targets {
    k8s {
        namespace: "prod"
        ingresses: ".*-lb"
    }
  }
  ```
  ```shell
  # Kube-DNS service
  targets {
    k8s {
        namespace: "kube-system"
        services: "kube-dns"
    }
  }
  ```
- `labelSelector`: Label based selector. It can be repeated, and works similar
  to the kubectl's --selector/-l flag. Example:
  ```shell
  targets {
    k8s {
        pods: ".*"
        labelSelector: "k8s-app"         # k8a-app label exists
        labelSelector: "role=frontend"   # label "role" is set to "frontend"
        labelSelector: "!no-monitoring"  # label "no-monitoring is not set"
    }
  }
  ```
- `portFilter`: (regex) Filter resources by port name or number (if port name is
  not set). This is useful for resources like endpoints and services, where each
  resource may have multiple ports. Example:
  ```shell
  targets {
    k8s {
        endpoints: ".*-service"
        portFilter: "http-.*"
    }
  }
  ```

### Cluster Resources Access

Note: If you've installed Cloudprober using
[Helm Chart](https://artifacthub.io/packages/helm/cloudprober/cloudprober), this
step is automatically taken care of.

Cloudprober discovers k8s resources using kubernetes APIs. It assumes that we
are interested in the cluster we are running it in, and uses in-cluster config
to talk to the kubernetes API server. For this set up to work, we need to give
our container read-only access to kubernetes resources:

```yaml
# Define a ClusterRole (resource-reader) for read-only access to the cluster
# resources and bind this ClusterRole to the default service account.

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloudprober
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  name: resource-reader
  namespace: default
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list"]
- apiGroups:
  - extensions
  - "networking.k8s.io" # k8s 1.14+
  resources:
  - ingresses
  - ingresses/status
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
 name: default-resource-reader
 namespace: default
subjects:
- kind: ServiceAccount
  name: cloudprober
  namespace: default
roleRef:
 kind: ClusterRole
 name: resource-reader
 apiGroup: rbac.authorization.k8s.io
EOF
```

This will create a new service account `cloudprober` and will give it read-only
access to the cluster resources.
