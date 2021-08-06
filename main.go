package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	metrics2 "github.com/anodot/anodot-common/pkg/metrics"
	anodotPrometheus "github.com/anodot/anodot-remote-write/pkg/prometheus"
	"github.com/anodot/anodot-remote-write/pkg/relabling"
	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "k8s.io/klog/v2"
)

const (
	DEFAULT_PORT              = 1234
	DEFAULT_NUMBER_OF_WORKERS = 20
	DEFAULT_TOKEN             = ""
	DEFAULT_ANODOT_URL        = "https://api.anodot.com"
)

var (
	kubernetesPodsFetchFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_kubernetes_pods_config_fetch_failed",
		Help: "Number of times pods relabel configuration was failed to fetch",
	})
)

func main() {
	var serverUrl = flag.String("url", DEFAULT_ANODOT_URL, "Anodot server url. Example: 'https://api.anodot.com'")
	var tokenFlagValue = flag.String("token", DEFAULT_TOKEN, "Account API Token")
	var serverPort = flag.Int("sever", DEFAULT_PORT, "Prometheus Remote Port")
	var maxWorkers = flag.Int64("workers", DEFAULT_NUMBER_OF_WORKERS, "Remote Write Workers -> Anodot")
	var filterOut = flag.String("filterOut", "", "Set an expression to remove metrics from stream")
	var filterIn = flag.String("filterIn", "", "Set an expression to add to stream")
	var murl = flag.String("murl", "", "Anodot Endpoint - Mirror")
	var mtoken = flag.String("mtoken", "", "Account AP Token - Mirror")
	var debug = flag.Bool("debug", false, "Print requests to stdout only")

	log.InitFlags(nil)
	err := flag.Set("v", defaultIfBlank(os.Getenv("ANODOT_LOG_LEVEL"), "3"))
	if err != nil {
		log.Fatal(err)
	}

	flag.Parse()
	token := envOrFlag("ANODOT_API_TOKEN", tokenFlagValue)

	log.Info(fmt.Sprintf("Anodot Remote Write version: '%s'. GitSHA: '%s'", version.VERSION, version.REVISION))
	log.V(3).Infof("Go Version: %s", runtime.Version())
	log.V(3).Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.V(3).Infof("Starting Anodot Remote Write on port: %d", *serverPort)

	var mirrorSubmitter metrics2.Submitter
	if *murl != "" {
		log.V(4).Infof("Anodot Address - Mirror: %s", *murl)
		log.V(4).Infof("Token - Mirror: %s", *mtoken)

		mirrorURL, err := url.Parse(*murl)
		if err != nil {
			log.Fatalf("Failed to construct Anodot server url with url=%q. Error:%s", *murl, err.Error())
		}

		mirrorSubmitter, err = metrics2.NewAnodot20Client(*mirrorURL, *mtoken, nil)
		if err != nil {
			log.Fatalf("Failed to create mirror submitter: %s", err.Error())
		}
	}

	tags := tags(os.Getenv("ANODOT_TAGS"))
	log.V(4).Infof("Metric tags: %s", tags)
	parser, err := anodotPrometheus.NewAnodotParser(filterIn, filterOut, tags)
	if err != nil {
		log.Fatalf("Failed to initialize Anodot parser. Error: %s", err.Error())
	}

	relabelConfigPath := os.Getenv("ANODOT_RELABEL_CONFIG_PATH")
	if len(strings.TrimSpace(relabelConfigPath)) > 0 {
		relabel, err := anodotPrometheus.NewMetricRelabel(relabelConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		parser.MetricsProcessors = append(parser.MetricsProcessors, relabel)
	}

	if len(strings.TrimSpace(os.Getenv("K8S_RELABEL_SERVICE_URL"))) > 0 {
		if err != nil {
			log.Fatalf("Failed to initialize k8s pod watcher. Error: %s", err.Error())
		}
		mapping, err := relabling.NewPodsMappingProvider(os.Getenv("K8S_RELABEL_SERVICE_URL"))
		if err != nil {
			log.Fatal(err)
		}

		err = mapping.UpdateConfig()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				log.V(4).Info("fetching pods mappings..")
				err := mapping.UpdateConfig()
				if err != nil {
					kubernetesPodsFetchFailed.Inc()
					log.Error(err)
				}
				time.Sleep(time.Second * 60)
			}
		}()
		parser.MetricsProcessors = append(parser.MetricsProcessors, &anodotPrometheus.KubernetesPodNameProcessor{PodsData: mapping})
	}

	primaryUrl, err := url.Parse(*serverUrl)
	if err != nil {
		log.Fatalf("Failed to construct Anodot server url with url=%q. Error:%s", *serverUrl, err.Error())
	}

	defaultTransport := http.DefaultTransport.(*http.Transport)
	defaultTransport.MaxIdleConnsPerHost = 1024
	defaultTransport.MaxIdleConns = 2048
	defaultTransport.IdleConnTimeout = 30 * time.Second
	client := &http.Client{
		Transport: defaultTransport,
		Timeout:   30 * time.Second,
	}

	primarySubmitter, err := metrics2.NewAnodot20Client(*primaryUrl, token, client)
	if err != nil {
		log.Fatalf("Failed to create Anodot metrics submitter: %s", err.Error())
	}

	//Actual server listening on port - serverPort
	var s = anodotPrometheus.Receiver{Port: *serverPort, Parser: parser}

	config, err := remote.NewWorkerConfig()
	if err != nil {
		log.Fatal("Failed to create worker config: ", err.Error())
	}

	if isFlagPassed("workers") {
		config.MaxWorkers = *maxWorkers
	}

	if isFlagPassed("debug") {
		config.Debug = *debug
	}

	primaryWorker, err := remote.NewWorker(primarySubmitter, config)
	if err != nil {
		log.Fatal("Failed to create worker: ", err.Error())
	}
	allWorkers := make([]*remote.Worker, 0)
	allWorkers = append(allWorkers, primaryWorker)

	if mirrorSubmitter != nil {
		mirrorWorker, err := remote.NewWorker(mirrorSubmitter, config)
		if err != nil {
			log.Fatal("Failed to create mirror worker: ", err.Error())
		}
		allWorkers = append(allWorkers, mirrorWorker)
	}

	ifReport := defaultIfBlank(os.Getenv("ANODOT_REPORT_MONITORING_METRICS"), "true")

	if ifReport != "false" {
		url, err := url.Parse(DEFAULT_ANODOT_URL)
		if err != nil {
			log.Fatalf("Could not parse Anodot url: %v", err)
		}

		p := defaultIfBlank(os.Getenv("ANODOT_MONTORING_REPORT_PERIOD_SEC"), "50")
		period, err := strconv.Atoi(p)
		if err != nil {
			log.Fatalf("Could not parse ANODOT_MONTORING_REPORT_PERIOD_SEC: %v", err)
		}

		monitoringSubmitter, err := metrics2.NewAnodot20Client(*url, token, nil)
		if err != nil {
			log.Fatalf("Failed to create monitoring submitter %v", err)
		}

		reporter := anodotPrometheus.NewReporter(monitoringSubmitter, *parser, period)
		reporter.Report()
	}

	accessKey := os.Getenv("ANODOT_ACCESS_KEY")
	ifSendToBC := defaultIfBlank(os.Getenv("ANODOT_SEND_TO_BC"), "true")
	sendToBCPeriod, err := strconv.Atoi(defaultIfBlank(os.Getenv("ANODOT_SEND_TO_BC_PERIOD_SEC"), "60"))
	if err != nil {
		log.Fatalf("Could not parse ANODOT_SEND_TO_BC_PERIOD_SEC: %v", err)
	}
	if ifSendToBC != "false" && len(strings.TrimSpace(accessKey)) > 0 {
		anodotPrometheus.SendAgentStatusToBC(primaryUrl, accessKey, token, sendToBCPeriod)
	}

	s.InitHttp(allWorkers)
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

func defaultIfBlank(actual string, fallback string) string {
	if len(strings.TrimSpace(actual)) == 0 {
		return fallback
	}
	return actual
}

func envOrFlag(envVarName string, flagValue *string) string {
	return defaultIfBlank(os.Getenv(envVarName), *flagValue)
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
