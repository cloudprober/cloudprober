local base_probe = {
  interval: '30s',
  timeout: '10s',
};

local kube_dns_probe = base_probe {
  name: 'kube_dns_kubernetes_default',
  type: 'DNS',
  targets: {
    k8s: {
      namespace: 'kube-system',
      services: 'kube-dns',
    },
  },
  dnsProbe: {
    resolvedDomain: 'kubernetes.default.svc.cluster.local',
    queryType: 'A',
  },
};

local vendor_tcp_probe = base_probe {
  name: 'vendor_tcp',
  type: 'TCP',
  targets: {
    hostNames: '1.2.3.4:8345',
  },
  tcpProbe: {},
};

local cloudprober_org_http_probe = base_probe {
  name: 'cloudprober_org_http',
  type: 'HTTP',
  targets: {
    hostNames: 'cloudprober.org',
  },
  validator: [
    {
      name: 'status_code_200',
      httpValidator: {
        successStatusCodes: '200',
      },
    },
  ],
  httpProbe: {
    protocol: 'HTTPS',
  },
};

{
  probe: [
    kube_dns_probe,
    vendor_tcp_probe,
    cloudprober_org_http_probe,
  ],
  surfacer: [
    {
      type: 'PROMETHEUS',
      prometheusSurfacer: {
        metricsPrefix: 'cloudprober_',
      },
    },
  ],
}
