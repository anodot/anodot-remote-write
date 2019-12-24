package prometheus

import (
	"fmt"
	anodotProm "github.com/anodot/anodot-common/pkg/metrics/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/anodot/anodot-remote-write/pkg/version"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"io/ioutil"
	"net/http"
)

type Receiver struct {
	Port   int
	Parser *anodotProm.AnodotParser
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
	log.Println(fmt.Sprintf("Initializing %d remote write config(s):", len(workers)), workers)

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

		for i := 0; i < len(workers); i++ {
			go func(n int) {
				workers[n].Do(data)
			}(i)
		}
	})

	http.HandleFunc(HEALTH_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/metrics", promhttp.Handler())

	log.Println(fmt.Sprintf("Application metrics available at '*:%d/metrics' ", rc.Port))

	versionInfo.With(prometheus.Labels{"version": version.VERSION, "git_sha1": version.REVISION}).Inc()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rc.Port), nil))

}
