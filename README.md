# Anodot Prometheous Remote Write

Anodot Prometheus Remote Write 

## Getting Started

git clone https://github.com/anodot/anodot-remote-write.git
cd anodot-remote-write 
GOOS=linux go build -o server main.go
docker build -t anodot-remote .
//push anodot-remote:<version> to repo 

//For K8s cluster deploy in Monitoring namespace:
Change deploy script as follows:
* . change image name in k8s Deployment
* . change  args: ["-url=https://api.anodot.com","-token=<font color="red">API TOKEN"</font>]
kubectl apply -f anodot-remote.yaml


### Prerequisites

Go 1.x 
Docker 1.18ce
Prometheus Server on K8s cluster

### Installing On K8s
On Prometheous Yaml add Remote Write to global settings, e.g.:

  prometheus.yml: |-
    global:
      scrape_interval: 60s
      evaluation_interval: 60s
    remote_write:
      - url: "http://anodot-prometheus.monitoring:1234/receive"

## Running the tests

Tests provided under anodotRemoteTests:
*  cd anodotRemoteTests 
*  go test .


## Authors

* **Yuval Dror** - *Initial work* 

## License and Disclaimer

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
