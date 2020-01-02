package main

import (
	"flag"
	"fmt"
	metrics2 "github.com/anodot/anodot-common/pkg/metrics"
	anodotPrometheus "github.com/anodot/anodot-common/pkg/metrics/prometheus"
	"net/url"
	"os"
	"strings"

	"github.com/anodot/anodot-remote-write/pkg/prometheus"
	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	log "github.com/sirupsen/logrus"
	"runtime"
)

func init() {
	level, e := log.ParseLevel(os.Getenv("ANODOT_LOG_LEVEL"))
	if e != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	log.Println("Application log level is:", log.GetLevel())
}

const (
	DEFAULT_PORT              = 1234
	DEFAULT_NUMBER_OF_WORKERS = 20
	DEFAULT_TOKEN             = ""
	DEFAULT_ANODOT_PORT       = "443"
	DEFAULT_ANODOT_URL        = "https://api.anodot.com"
)

func main() {
	var serverUrl = flag.String("url", DEFAULT_ANODOT_URL, "Anodot server url. Example: 'https://api.anodot.com'")
	var port = flag.String("port", DEFAULT_ANODOT_PORT, "Anodot server port. WARNING: will be removed in 2.x version")
	var token = flag.String("token", DEFAULT_TOKEN, "Account API Token")
	var serverPort = flag.Int("sever", DEFAULT_PORT, "Prometheus Remote Port")
	var workers = flag.Int64("workers", DEFAULT_NUMBER_OF_WORKERS, "Remote Write Workers -> Anodot")
	var filterOut = flag.String("filterOut", "", "Set an expression to remove metrics from stream")
	var filterIn = flag.String("filterIn", "", "Set an expression to add to stream")
	var murl = flag.String("murl", "", "Anodot Endpoint - Mirror")
	var mport = flag.String("mport", "", "Anodot Port - Mirror")
	var mtoken = flag.String("mtoken", "", "Account AP Token - Mirror")
	var debug = flag.Bool("debug", false, "Print requests to stdout only")

	flag.Parse()

	log.Info(fmt.Sprintf("Anodot Remote Write version: '%s'. GitSHA: '%s'", version.VERSION, version.REVISION))
	log.Debugf("Go Version: %s", runtime.Version())
	log.Debugf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.Debugf("Starting Anodot Remote Write on port: %d", *serverPort)

	//TODO vnekhai: move this to place where workers info is printed
	//log.Debugf(fmt.Sprintf("API Token: %q", *token))

	var mirrorSubmitter metrics2.Submitter
	if *murl != "" {
		log.Debug("Anodot Address - Mirror:", *murl, *mport)
		log.Debug("Token - Mirror:", *mtoken)

		mirrorURL, err := toUrl(murl, port)
		if err != nil {
			log.Fatalf("Failed to construct anodot server url with host=%q and port=%q. Error:%s", *murl, *mport, err.Error())
		}

		mirrorSubmitter, err = metrics2.NewAnodot20Submitter(mirrorURL.String(), *mtoken, nil)
		if err != nil {
			log.Fatalf("Failed to create mirror submitter: %s", err.Error())
		}
	}

	parser, err := anodotPrometheus.NewAnodotParser(filterIn, filterOut, tags(os.Getenv("ANODOT_TAGS")))
	if err != nil {
		log.Fatalf("Failed to initialize anodot parser. Error: %s", err.Error())
	}

	primaryUrl, err := toUrl(serverUrl, port)
	if err != nil {
		log.Fatalf("Failed to construct anodot server url with host=%q and port=%q. Error:%s", *serverUrl, *port, err.Error())
	}

	primarySubmitter, err := metrics2.NewAnodot20Submitter(primaryUrl.String(), *token, nil)
	if err != nil {
		log.Fatalf("Failed to create Anodot metrics submitter: %s", err.Error())
	}

	//Actual server listening on port - serverPort
	var s = prometheus.Receiver{Port: *serverPort, Parser: parser}

	//Initializer -> listener, endpoints etc
	primaryWorker, err := remote.NewWorker(primarySubmitter, *workers, *debug)
	if err != nil {
		log.Fatal("Failed to create worker: ", err.Error())
	}

	allWorkers := make([]*remote.Worker, 0)
	allWorkers = append(allWorkers, primaryWorker)

	if mirrorSubmitter != nil {
		mirrorWorker, err := remote.NewWorker(mirrorSubmitter, *workers, *debug)
		if err != nil {
			log.Fatal("Failed to create mirror worker: ", err.Error())
		}
		allWorkers = append(allWorkers, mirrorWorker)
	}
	s.InitHttp(allWorkers)
}

//TODO remove in future
func toUrl(serverUrl *string, port *string) (*url.URL, error) {
	return url.Parse(*serverUrl + ":" + *port)
}

func tags(envVar string) map[string]string {
	res := make(map[string]string)

	split := strings.Split(envVar, ";")
	for _, v := range split {
		if strings.Contains(v, "=") {
			kv := strings.Split(v, "=")
			res[kv[0]] = kv[1]
		}
	}
	return res
}
