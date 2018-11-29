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
*  cd anodotRemoteTests 
*  go test .


## Disclaimer

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

## Authors

* **Yuval Dror** - *Initial work* 

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
