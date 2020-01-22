[![Latest Release](https://img.shields.io/github/release/anodot/anodot-remote-write.svg)](https://github.com/anodot/anodot-remote-write/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/anodot/anodot-remote-write)](https://goreportcard.com/report/github.com/anodot/anodot-remote-write)
[![Docker pulls](https://img.shields.io/docker/pulls/anodot/prometheus-remote-write.svg)](https://hub.docker.com/r/anodot/prometheus-remote-write)
[![codecov](https://codecov.io/gh/anodot/anodot-remote-write/branch/master/graph/badge.svg)](https://codecov.io/gh/anodot/anodot-remote-write)

# Anodot Prometheous Remote Write

`anodot-prometheus-remote-write` is a service which receives [Prometheus](https://github.com/prometheus) metrics through [`remote_write`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write), converts metrics and sends them into [Anodot](https://www.anodot.com).

* [Building application](#building-application)
  * [Prerequisites](#prerequisites)
  * [Running tests](#running-tests)
* [Deploying application application](#deploying-application-application)
  * [Prerequisites](#prerequisites-1)
     * [Using helm](#using-helm)
     * [Using docker-compose](#using-docker-compose)
* [Configuring Prometheus server](#configuring-prometheus-server)
  * [Authors](#authors)
  * [License and Disclaimer](#license-and-disclaimer)

# Building From Source
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

# Deploy 

Optional Configuration options. Should be specified under `configuration.env` section in values.yaml

| Env variable                | Description                                                                   | Default       | 
| ----------------------------|-------------------------------------------------------------------------------| --------------|
| ANODOT_LOG_LEVEL            | Application log level. Supported options are: `panic, fatal, error, warning, info, debug, trace`| info          |
| ANODOT_TAGS                 | Format `TAG1=VALUE1;TAG2=VALUE2` Static tags that will be added to Anodot |["source":"prometheus-remote-write"]|
| ANODOT_HTTP_DEBUG_ENABLED   | Should be used to enable HTTP requests/response dumps to stdout |false|

## Prerequisites
- Docker 1.18ce.
- Prometheus Server

### Using Helm

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

# Configuring Prometheus Server
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
