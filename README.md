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
### Kubernetes pod names changes

*Problem*
Kubernetes pods which are managed by Deployments and Replicasets has unique random names.
For example:
```bash
cloudwatch-exporter-945b6685d-hxfcz                          1/1     Running   0          123d
elastic-exporter-6c476798f7-6xgls                            1/1     Running   0          14d
``` 

This will create next metric in Prometheus 
```bash
container_memory_usage_bytes{anodot_include="true",container_name="POD",image="k8s.gcr.io/pause-amd64:3.0",job="kubelet",namespace="default",node="ip-10-0-37-203.ap-southeast-2.compute.internal",pod_name="anodotd-webapp-b57c79d8-fctnf"}
```

Each time deployment pods are created - random names are generated, and metrics history is lost in Anodot server


**Solution**
Each pods in deployment/replicaset is assigned with unique label `anodot.com/podName=${deployment-name}-${ordinal}`, where ordinal is incrementally assigned to each pod.
When metrics arrives to anodot-prometheus-remote write, original `pod` and `pod_name` is replaced with `anodot.com/podName` value. 

If there is no value for given pod, metric is dropped until pods cache is updated (This happens each 60s.)

Lets take a look at next example:

1.
```bash
kubectl get pods

NAME                                                         READY   STATUS    RESTARTS   AGE
    cloudwatch-exporter-945b6685d-hxfcz                          1/1     Running   0          124d
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
Pods cache is updated each 60s so there is a chance that some metrics will be dropped.


helm11 upgrade -i anodot-pod-relabel --namespace=monitoring .