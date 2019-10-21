[![Go Report Card](https://goreportcard.com/badge/github.com/anodot/anodot-remote-write)](https://goreportcard.com/report/github.com/anodot/anodot-remote-write)
[![Docker Pulls](https://img.shields.io/docker/pulls/anodot/prometheus-remote-write)][https://hub.docker.com/r/anodot/prometheus-remote-write]

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

# Building application
## Prerequisites
 - Go >= 1.11
 
```shell script
git clone https://github.com/anodot/anodot-remote-write.git && cd anodot-remote-write
make all
```

## Running tests
```shell script
make test
```

# Deploying application application

## Prerequisites
- Docker >=1.18ce
- Prometheus Server

### Using helm
1. `helm repo add anodot https://anodot.github.io/helm-charts`
2. `helm fetch anodot/anodot-prometheus-remote-write --untar`
3. Edit `values.yaml` and fill in required variables.
4. Run next command:

```shell script
helm upgrade -i anodot-remote-write -n monitoring . 
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
  - url: "http://anodot-prometheus-remote-write:1234/receive"
  version: v2.10.0
```

More Prometheus configuration options available on [this](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md#remotewritespec) page.

## Authors

* **Yuval Dror** - *Initial work* 

## License and Disclaimer

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
