package prometheus

import (
	"strconv"
	"time"

	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
)

var (
	monitoringAnodotServerResponseTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "anodot_monitoring_server_response_time_seconds",
		Help:       "Anodot server response time in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"anodot_url"})

	monitoringErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_monitoring_metrics_errors",
		Help: "Total number of errors occurred while collecting monitoring metrics",
	})

	monitoringAnodotSubmitterErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_monitoring_metrics_submission_errors",
		Help: "Total number of errors occurred while sending metrics to Anodot api",
	}, []string{"anodot_url"})

	monitoringServerHTTPResponses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_monitoring_server_http_responses_total",
		Help: "Total number of HTTP responses of Anodot server",
	}, []string{"anodot_url", "response_code"})
)

// Reporter used for reporting remote-write metrics
// to agents endpoint
type Reporter struct {
	metricsSubmitter *metrics.Anodot20Client
	parser           AnodotParser
}

func NewReporter(submitter *metrics.Anodot20Client, parser AnodotParser) *Reporter {
	return &Reporter{submitter, parser}
}

func (r *Reporter) pushMetrics(metricsToSend []metrics.Anodot20Metric) {
	ts := time.Now()

	anodotResponse, err := r.metricsSubmitter.SubmitMonitoringMetrics(metricsToSend)
	if anodotResponse != nil && anodotResponse.RawResponse() != nil {
		monitoringServerHTTPResponses.WithLabelValues(r.metricsSubmitter.AnodotURL().Host, strconv.Itoa(anodotResponse.RawResponse().StatusCode)).Inc()
	}
	if err != nil {
		monitoringAnodotSubmitterErrors.WithLabelValues(r.metricsSubmitter.AnodotURL().Host).Inc()
		log.Error("Failed to send metrics: ", err)
		return
	}

	monitoringAnodotServerResponseTime.WithLabelValues(r.metricsSubmitter.AnodotURL().Host).Observe(time.Since(ts).Seconds())
}

func (r *Reporter) Report() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				samples, err := getSamples()
				if err != nil {
					log.Errorf("Failed to collect monitoring metrics %v", err)
					monitoringErrors.Add(1)
				}
				data := r.parseMetrics(samples)
				r.pushMetrics(data)
			}
		}
	}()

}

func (r *Reporter) parseMetrics(samples []*model.Sample) []metrics.Anodot20Metric {
	return r.parser.ParsePrometheusRequest(samples)
}

func getSamples() ([]*model.Sample, error) {
	samples := make([]*model.Sample, 0)

	rf := prometheus.DefaultGatherer
	mfs, err := rf.Gather()
	if err != nil {
		return samples, err
	}
	vec, err := expfmt.ExtractSamples(&expfmt.DecodeOptions{
		Timestamp: model.Now(),
	}, mfs...)

	if err != nil {
		return samples, err
	}

	for _, m := range vec {
		samples = append(samples, m)
	}
	return samples, nil
}
