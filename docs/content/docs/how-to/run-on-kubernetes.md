---
menu:
  docs:
    parent: "how-to"
    weight: 15
title: "Running On Kubernetes"
date: 2022-11-01T17:24:32-07:00
---

Kubernetes is a popular platform for running containers, and Cloudprober
container runs on Kubernetes right out of the box. This document shows how you
can run Cloudprober on kubernetes, use ConfigMap for config, and discover
kubernetes targets automatically.

{{< alert context=info icon=ⓘ >}} If you use helm charts for k8s installations,
[Cloudprober helm chart](https://github.com/cloudprober/helm-charts) provides
the most convenient way to run Cloudprober on k8s. {{< /alert >}}

## ConfigMap

In Kubernetes, a convenient way to provide config to containers is to use config
maps. Let's create a config that specifies a probe to monitor "google.com".

```bash
probe {
  name: "google-http"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }
  http_probe {}
  interval_msec: 15000
  timeout_msec: 1000
}
```

Save this config in `cloudprober.cfg`, create a config map using the following
command:

```bash
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg
```

If you change the config, you can update the config map using the following
command:

```bash
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg  -o yaml --dry-run | \
  kubectl replace -f -
```

## Deployment Map

Now let's add a `deployment.yaml` to add the config volume and cloudprober
container:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudprober
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloudprober
  template:
    metadata:
      annotations:
        checksum/config: "${CONFIG_CHECKSUM}"
      labels:
        app: cloudprober
    spec:
      volumes:
        - name: cloudprober-config
          configMap:
            name: cloudprober-config
      containers:
        - name: cloudprober
          image: cloudprober/cloudprober
          command: ["/cloudprober"]
          args: ["--config_file", "/cfg/cloudprober.cfg"]
          volumeMounts:
            - name: cloudprober-config
              mountPath: /cfg
          ports:
            - name: http
              containerPort: 9313
          livenessProbe:
            httpGet:
              path: /health
              port: 9313
          readinessProbe:
            httpGet:
              path: /health
              port: 9313
---
apiVersion: v1
kind: Service
metadata:
  name: cloudprober
  labels:
    app: cloudprober
spec:
  ports:
    - port: 9313
      protocol: TCP
      targetPort: 9313
  selector:
    app: cloudprober
  type: NodePort
```

Note that we added an annotation to the deployment spec; this annotation allows
us to update the deployment whenever cloudprober config changes. We can update
this annotation based on the local cloudprober config content, and update the
deployment using the following one-liner:

```shell
# Update the config checksum annotation in deployment.yaml before running
# kubectl apply.
export CONFIG_CHECKSUM=$(kubectl get cm/cloudprober-config -o yaml | sha256sum) && \
cat deployment.yaml | envsubst | kubectl apply -f -
```

(Note: If you use Helm for Kubernetes deployments, Helm provides
[a more native way](https://helm.sh/docs/howto/charts_tips_and_tricks/#automatically-roll-deployments)
to include config checksums in deployments.)

Applying the above yaml file, should create a deployment with a service at port
9313:

```bash
$ kubectl get deployment
NAME          READY   UP-TO-DATE   AVAILABLE   AGE
cloudprober   1/1     1            1           94m

$ kubectl get service cloudprober
NAME          TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
cloudprober   NodePort   10.31.249.108   <none>        9313:31367/TCP   94m
```

Now you should be able to access various cloudprober URLs (`/status` for
status,`/config` for config, `/metrics` for prometheus-format metrics) from
within the cluster. For quick verification you can also set up a port forwarder
and access these URLs locally at `localhost:9313`:

```shell
kubectl port-forward svc/cloudprober 9313:9313
```

Once you've verified that everything is working as expected, you can go on
setting up metrics collection through prometheus (or stackdriver) in usual ways.

## Kubernetes Targets

If you're running on Kubernetes, you'd probably want to monitor Kubernetes
resources (e.g. pods, endpoints, etc) as well. Cloudprober supports [dynamic
discovery]({{< ref targets.md >}}#dynamically-discovered-targets) of Kubernetes
resources through the targets type
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

See [Kubernetes Targets]({{< ref k8s_targets.md >}}#kubernetes-targets) for more
details on Kubernetes targets.

## Push Config Update

To push new cloudprober config to the cluster:

```bash
# Update the config map
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg  -o yaml --dry-run | \
  kubectl replace -f -

# Update deployment
export CONFIG_CHECKSUM=$(kubectl get cm/cloudprober-config -o yaml | sha256sum) && \
cat deployment.yaml | envsubst | kubectl apply -f -
```

Cloudprober should now start monitoring cloudprober endpoints. To verify:

```bash
# Set up port fowarding such that you can access cloudprober:9313 through
# localhost:9313.
kubectl port-forward svc/cloudprober 9313:9313 &

# Check status
curl localhost:9313/status

# Check metrics (prometheus data format)
curl localhost:9313/metrics
```

If you're running on GKE and have not disabled cloud logging, you'll also see
logs in
[Stackdriver Logging](https://pantheon.corp.google.com/logs/viewer?resource=gce_instance).
