# Anodot Prometheous Remote Write

`anodot-prometheus-remote-write` is a service which receives [Prometheus](https://github.com/prometheus) metrics through [`remote_write`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write), converts metrics and sends them into [Anodot](https://www.anodot.com).

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
- Docker 1.18ce.
- Prometheus Server on K8s cluster.

### Using helm

1. Navigate to `deployment/helm/anodot-remote-write`
2. Edit `values.yaml` and fill in required variables.
3. Run next command:

```shell script
helm upgrade -i anodot-remote-write -n monitoring . 
```
This command will install application in `monitoring` namespace.

### Using kubernetes config files
TODO

# Configuring Prometheus server
In Prometheus configuration file (default `prometheus.yml`), add `remote_write` [configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write)
 ```yaml
 global:
      scrape_interval: 60s
      evaluation_interval: 60s
    remote_write:
      - url: "http://anodot-prometheus.monitoring:1234/receive"
```

## Authors

* **Yuval Dror** - *Initial work* 

## License and Disclaimer

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
