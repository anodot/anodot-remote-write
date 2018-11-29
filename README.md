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
* . change  args: ["-url=https://api.anodot.com","-token=<API TOKEN>"]
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
* . cd anodotRemoteTests 
* . go test .


## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/your/project/tags). 

## Authors

* **Billie Thompson** - *Initial work* - [PurpleBooth](https://github.com/PurpleBooth)

See also the list of [contributors](https://github.com/your/project/contributors) who participated in this project.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* Hat tip to anyone whose code was used
* Inspiration
* etc




