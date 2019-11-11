package main

import (
	"flag"
	"fmt"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"github.com/anodot/anodot-common/remoteStats"
	"github.com/anodot/anodot-remote-write/pkg/prometheus"
	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	"github.com/rcrowley/go-metrics"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	FLUSH_STATISTICS_TO_LOG   = 300
	DEFAULT_PORT              = 1234
	DEFAULT_NUMBER_OF_WORKERS = 20
	DEFAULT_TOKEN             = ""
	DEFAULT_ANODOT_PORT       = "443"
	DEFAULT_ANODOT_URL        = "https://api.anodot.com"
)

func Mb(b uint64) uint64 {
	return b / 1024 / 1024
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", Mb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", Mb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", Mb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func main() {

	var url = flag.String("url", DEFAULT_ANODOT_URL, "Anodot Endpoint")
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

	if *token == "" {
		log.Println("No Token provided")
		flag.Usage()
		os.Exit(1)
	}

	log.Println("---Anodot Remote Write---")

	log.Printf("App version: %q. GitSHA: %q", version.VERSION, version.REVISION)
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.Println("Starting Anodot Remote Port: ", *serverPort)
	log.Println("Anodot Address:", *url, *port)
	log.Println("Token:", *token)
	if *murl != "" {
		log.Println("Anodot Address - Mirror:", *murl, *mport)
		log.Println("Token - Mirror:", *mtoken)
		log.Println("Number of Workers:", *workers)
	}

	//Prepare filters
	err, parser := anodotParser.NewAnodotParser(filterIn, filterOut)

	if err != nil {
		log.Println(err.Error())
	}

	//Initialize statistics
	var Stats = remoteStats.NewStats()
	go metrics.Log(Stats.Registry, FLUSH_STATISTICS_TO_LOG*time.Second, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))

	//Print Memory Statistics
	go func() {
		for range time.Tick(5 * time.Minute) {
			PrintMemUsage()
		}
	}()

	submitter := anodotSubmitter.NewAnodot20Submitter(*url, *port, *token, &Stats, *murl, *mport, *mtoken)

	//Workers manager -> number of threads that communicate with Anodot
	var w = remote.NewWorker(*workers, *debug)

	//Actual server listening on port - serverPort
	var s = prometheus.Receiver{Port: *serverPort, Parser: &parser}

	//Initializer -> listener, endpoints etc
	s.InitHttp(&submitter, &Stats, &w)

}
