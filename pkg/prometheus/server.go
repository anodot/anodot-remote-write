package prometheus

import (
	"fmt"
	"os"
	"time"

	"github.com/anodot/anodot-remote-write/utils"
	log "k8s.io/klog/v2"

	"io/ioutil"
	"net/http"

	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
)

type Receiver struct {
	Port   int
	Parser *AnodotParser
}

var (
	totalRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_remote_write_received_requests",
		Help: "The total number of received requests from Prometheus server",
	})

	httpResponses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_remote_write_http_responses_total",
		Help: "Total number of Anodot Remote Write HTTP responses",
	}, []string{"response_code"})

	versionInfo = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_remote_write_version",
		Help: "Build info",
	}, []string{"version", "git_sha1"})
	r = prometheus.NewRegistry()
)

const RECEIVER_ENDPOINT = "/receive"
const HEALTH_ENDPOINT = "/health"

func (rc *Receiver) protoToSamples(req *prompb.WriteRequest) model.Samples {
	var samples model.Samples
	for _, ts := range req.Timeseries {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}

		for _, s := range ts.Samples {
			samples = append(samples, &model.Sample{
				Metric:    metric,
				Value:     model.SampleValue(s.Value),
				Timestamp: model.Time(s.Timestamp),
			})
		}
	}
	return samples
}

func (rc *Receiver) InitHttp(workers []*remote.Worker) {
	log.V(2).Infof("Initializing %d remote write config(s): %s", len(workers), workers)

	if os.Getenv("ANODOT_PUSH_METRICS_ENABLED") == "true" {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			quit := make(chan struct{})

			for {
				select {
				case <-ticker.C:
					samples, err := utils.FetchMetrics("http://127.0.0.1:1234/metrics", 3, time.Second*5)
					if err != nil {
						log.Errorf("failed to scrape own metrics endpoint. %s", err.Error())
					}

					for i := 0; i < len(workers); i++ {
						workers[i].Do(rc.Parser.ParsePrometheusRequest(samples))
					}
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}

	http.HandleFunc(RECEIVER_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		totalRequests.Inc()

		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			httpResponses.With(prometheus.Labels{"response_code": "500"}).Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			httpResponses.With(prometheus.Labels{"response_code": "400"}).Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			httpResponses.With(prometheus.Labels{"response_code": "400"}).Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data := rc.Parser.ParsePrometheusRequest(rc.protoToSamples(&req))
		if len(data) == 0 {
			return
		}

		for i := 0; i < len(workers); i++ {
			workers[i].Do(data)
		}
	})

	http.HandleFunc(HEALTH_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/metrics", promhttp.Handler())
	log.V(2).Infof("Application metrics available at '*:%d/metrics' ", rc.Port)

	versionInfo.With(prometheus.Labels{"version": version.VERSION, "git_sha1": version.REVISION}).Inc()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rc.Port), nil))
}
