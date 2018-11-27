package main

import (
	"github.com/anodot/anodot-remote-write/anodotServer"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"flag"
	"github.com/anodot/anodot-remote-write/remoteStats"
	"time"
	"os"
	"log"
	"fmt"
	"runtime"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/rcrowley/go-metrics"
)

const (
	FLUSH_STATISTICS_TO_LOG  = 5
	DEFAULT_PORT = 1234
	DEFAULT_NUMBER_OF_WORKERS = 20
	DEFAULT_TOKEN = ""
	DEFAULT_ANODOT_PORT = "443"
	DEFAULT_ANODOT_URL = "https://api.anodot.com"
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
	var token = flag.String("token", DEFAULT_TOKEN, "Account AP Token")
	var serverPort = flag.Int("sever", DEFAULT_PORT, "Prometheus Remote Port")
	var workers = flag.Int64("workers", DEFAULT_NUMBER_OF_WORKERS, "Remote Write Workers -> Anodot")
	var filterOut = flag.String("filterOut","","Set an expression to remove metrics from stream")
	var filterIn = flag.String("filterIn","","Set an expression to add to stream")

	flag.Parse()

	if(*token == ""){
		log.Println("No Token provided")
		flag.Usage()
	}

	log.Println("Starting Anodot Remote Port: ",*serverPort)
	log.Println("Anodot Address:",*url,*port)
	log.Println("Token:",*token)
	log.Println("Number of Workers:",*workers)

	//Prepare filters
	err,parser := anodotParser.NewAnodotParser(filterIn,filterOut)

	if(err != nil){
		log.Println(err.Error())
	}

	//Initialize statistics
	var Stats = remoteStats.NewStats()
	go metrics.Log(Stats.Registry, FLUSH_STATISTICS_TO_LOG * time.Second, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))

	//Print Memory Statistics
	go func() {
		for _ = range time.Tick(5 * time.Minute) {
			PrintMemUsage();
		}
	}()

	//Submitter sends parsed metrics to Anodot
	submitter := anodotSubmitter.NewAnodot20Submitter(*url,*port,*token,&Stats)

	//Workers manager -> number of threads that communicate with Anodot
	var w =anodotServer.NewWorker(*workers,&Stats)

	//Actual server listening on port - serverPort
	var s = anodotServer.Receiver{Port:*serverPort,Parser:&parser}

	//Initializer -> listener, endpoints etc
	s.InitHttp(&submitter,&Stats,&w)

}