[![Latest Release](https://img.shields.io/github/release/anodot/anodot-remote-write.svg)](https://github.com/anodot/anodot-remote-write/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/anodot/anodot-remote-write)](https://goreportcard.com/report/github.com/anodot/anodot-remote-write)
[![Docker pulls](https://img.shields.io/docker/pulls/anodot/prometheus-remote-write.svg)](https://hub.docker.com/r/anodot/prometheus-remote-write)
[![codecov](https://codecov.io/gh/anodot/anodot-remote-write/branch/master/graph/badge.svg)](https://codecov.io/gh/anodot/anodot-remote-write)

# Anodot Prometheous Remote Write

`anodot-prometheus-remote-write` is a service which receives [Prometheus](https://github.com/prometheus) metrics through [`remote_write`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write), converts metrics and sends them into [Anodot](https://www.anodot.com).

* [Building application](#building-application)
  * [Prerequisites](#prerequisites)
  * [Running tests](#running-tests)
* [Deploying application application](#deploying-application)
  * [Prerequisites](#prerequisites-1)
     * [Using helm](#using-helm)
     * [Using docker-compose](#using-docker-compose)
* [Configuring Prometheus server](#configuring-prometheus-server)
  * [Authors](#authors)
  * [License and Disclaimer](#license-and-disclaimer)

# Building application
## Prerequisites
 - Go >= 1.13
 
```shell script
git clone https://github.com/anodot/anodot-remote-write.git && cd anodot-remote-write
make all
```

## Running tests
```shell script
make test
```

# Deploying application

Optional Configuration options. Should be specified under `configuration.env` section in values.yaml

| Env variable                | Description                                                                   | Default       | 
| ----------------------------|-------------------------------------------------------------------------------| --------------|
| ANODOT_LOG_LEVEL            | Application log level. Supported options are: `1, 2, 3, 4, 5`| info          |
| ANODOT_TAGS                 | Format `TAG1=VALUE1;TAG2=VALUE2` Static tags that will be added to Anodot |["source":"prometheus-remote-write"]|
| ANODOT_HTTP_DEBUG_ENABLED   | Should be used to enable HTTP requests/response dumps to stdout |false|

## Prerequisites
- Docker 1.18ce.
- Prometheus Server

### Using helm

```shell script
helm repo add anodot https://anodot.github.io/helm-charts
```

```shell script
helm fetch anodot/anodot-prometheus-remote-write --untar
```

Navigate to `anodot-prometheus-remote-write` folder and edit `values.yaml`with required values (image version, anodot token,
etc)

Run next command to install chart
```shell script
helm upgrade -i anodot-remote-write . --namespace=monitoring
```

This command will install application in `monitoring` namespace.

### Using docker-compose

```shell script
cd deployment/docker-compose
```
Open `docker-compose.yaml` and edit if needed, specifying required configuration parameters.
Run next command to start application:
```shell script
docker-compose up -d 
``` 

# Configuring Prometheus server
In Prometheus configuration file (default `prometheus.yml`), add `remote_write` [configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write)
 ```yaml
 global:
      scrape_interval: 60s
      evaluation_interval: 60s
    remote_write:
      - url: "http://anodot-prometheus-remote-write:1234/receive"
        metric_relabel_configs:
        - source_labels: [ __name__ ]
          regex: '(nginx_ingress_controller_requests|nginx_ingress_controller_ingress_upstream_latency_seconds)'
          action: drop
```

[Prometheus operator](https://github.com/coreos/prometheus-operator) configuration example (some fields omitted for clarity):
```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  labels:
    app: prometheus-operator-prometheus
  name: prometheus
spec:
    remoteWrite:
     - url: http://anodot-prometheus-remote-write:1234/receive
     #  writeRelabelConfigs:
     #    - action: drop
     #      regex: '(apiserver_request_count|prometheus_remote_storage_sent_batch_duration_seconds_bucket)'
     #      sourceLabels: [__name__]
       queueConfig:
         maxSamplesPerSend: 1000
  version: v2.10.0
```

More Prometheus configuration options available on [this](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#remotewritespec) page.

## Authors

* **Yuval Dror** - *Initial work* 

## License and Disclaimer

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details


## Advanced features
### Re-writing kubernetes pod name

*Problem*

Kubernetes pods which are managed by Deployments and Replicasets has unique random names, each time pod is re-created - new random name is assigned to pod.

For example:
```bash
cloudwatch-exporter-945b6685d-hxfcz       1/1     Running   0  123d
elastic-exporter-6c476798f7-6xgls         1/1     Running   0  14d
``` 

Prometheus metrics for such pod will look like this:
```bash
container_memory_usage_bytes{job="kubelet",namespace="monitoring",node="cluster-node1",pod_name="elastic-exporter-6c476798f7-6xgls"}
container_memory_usage_bytes{job="kubelet",namespace="monitoring",node="cluster-node2",pod_name="cloudwatch-exporter-945b6685d-hxfcz"}
```

TODO: explain why this is bad
Changing pod name label, will cause new metric creating in Anodot system. 


*Solution*

Each pods in deployment/replicaset/daemonset is assigned with unique label `anodot.com/podName=${deployment-name}-${ordinal}`, where ordinal is incrementally assigned to each pod.
When metrics arrives to anodot-prometheus-remote write, original `pod` and `pod_name` is replaced with `anodot.com/podName` value.

anodot-prometheus-remote-write application, keeps track of all pods information (mapping between original pod name and pod name under `anodot.com/podName` label)
If there are no mapping information for given pod, **such metrics will not be sent** to Anodot system to prevent from creating unnecessary metrics.

Lets take a look at next example on how re-writing is done:

1.
```bash
kubectl get pods

NAME                                          READY   STATUS    RESTARTS   AGE
    cloudwatch-exporter-945b6685d-hxfcz       1/1     Running   0          124d
```
2.
```bash
kubectl describe pods cloudwatch-exporter-945b6685d-hxfcz
Name:           cloudwatch-exporter-945b6685d-hxfcz
Namespace:      monitoring
Labels:         anodot.com/podName=cloudwatch-exporter-0
                pod-template-hash=501622418
```
3. 
Kubelet metrics scraped by Prometheus looks like this:
```bash
container_memory_usage_bytes{container_name="elastic-seporter",job="kubelet",namespace="monitoring",pod_name="cloudwatch-exporter-945b6685d-hxfcz"}	
```
4. After anodot-prometheus-remote-write process this metrics, `pod_name` value will be changed to `cloudwatch-exporter-0` before sending metrics to Anodot system.


**Important notes**
 - Pods cache is updated each 60s so there is a chance that some metrics will be dropped. (`anodot_parser_kubernetes_relabling_metrics_dropped` represent total number of dropped metrics)
 
**Enabling feature**
1. Install anodot-pod-relabel helm chart by following steps here https://github.com/anodot/helm-charts/tree/master/charts/anodot-pod-relabel
2. set `K8S_RELABEL_SERVICE_URL` under `Values.configuration.env` in anodot-remote-write helm chart and proceed with installation