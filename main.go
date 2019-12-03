package main

import (
	"flag"
	metrics2 "github.com/anodot/anodot-common/pkg/metrics"
	anodotPrometheus "github.com/anodot/anodot-common/pkg/metrics/prometheus"
	"net/url"

	"github.com/anodot/anodot-remote-write/pkg/prometheus"
	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	"log"
	"runtime"
)

const (
	DEFAULT_PORT              = 1234
	DEFAULT_NUMBER_OF_WORKERS = 20
	DEFAULT_TOKEN             = ""
	DEFAULT_ANODOT_PORT       = "443"
	DEFAULT_ANODOT_URL        = "https://api.anodot.com"
)

func main() {
	var serverUrl = flag.String("url", DEFAULT_ANODOT_URL, "Anodot Endpoint")
	var port = flag.String("port", DEFAULT_ANODOT_PORT, "Anodot Port")
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

	log.Println("---Anodot Remote Write---")

	log.Printf("App version: %q. GitSHA: %q", version.VERSION, version.REVISION)
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.Println("Starting Anodot Remote Port: ", *serverPort)
	log.Println("Token:", *token)

	var mirrorSubmitter metrics2.Submitter
	if *murl != "" {
		log.Println("Anodot Address - Mirror:", *murl, *mport)
		log.Println("Token - Mirror:", *mtoken)
		log.Println("Number of Workers:", *workers)

		mirrorURL, err := toUrl(murl, port)
		if err != nil {
			log.Fatalf("[ERROR]: failed to construct anodot server url with host=%q and port=%q. Error:%s", *murl, *mport, err.Error())
		}

		mirrorSubmitter, err = metrics2.NewAnodot20Submitter(mirrorURL.String(), *mtoken, nil)
		if err != nil {
			log.Fatalf("[ERROR]: failed to create mirror submitter: %s", err.Error())
		}
	}

	parser, err := anodotPrometheus.NewAnodotParser(filterIn, filterOut)
	if err != nil {
		log.Fatalf("[ERROR]: failed to initialize anodot parser. Error: %s", err.Error())
	}

	primaryUrl, err := toUrl(serverUrl, port)
	if err != nil {
		log.Fatalf("[ERROR]: failed to construct anodot server url with host=%q and port=%q. Error:%s", *serverUrl, *port, err.Error())
	}

	primarySubmitter, err := metrics2.NewAnodot20Submitter(primaryUrl.String(), *token, nil)
	if err != nil {
		log.Fatalf("[ERROR]: failed to create Anodot metrics submitter: %s", err.Error())
	}

	//Actual server listening on port - serverPort
	var s = prometheus.Receiver{Port: *serverPort, Parser: parser}

	//Initializer -> listener, endpoints etc
	primaryMapping := prometheus.AnodotApiMapping{
		Workers:   remote.NewWorker(*workers, *debug),
		Submitter: primarySubmitter,
	}

	mappings := make([]prometheus.AnodotApiMapping, 0)
	mappings = append(mappings, primaryMapping)

	if mirrorSubmitter != nil {
		mappings = append(mappings, prometheus.AnodotApiMapping{
			Workers:   remote.NewWorker(*workers, *debug),
			Submitter: mirrorSubmitter,
		})
	}

	s.InitHttp(mappings)
}

//TODO remove in future
func toUrl(serverUrl *string, port *string) (*url.URL, error) {
	return url.Parse(*serverUrl + ":" + *port)
}
