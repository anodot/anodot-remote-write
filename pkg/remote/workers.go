package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var metricsPerRequest = 1000
var maxMetricsBufferSize = 10000

type Worker struct {
	metricsSubmitter metrics.Submitter

	Max           int64
	Current       int64
	MetricsBuffer []metrics.Anodot20Metric
	Debug         bool
}

func (w *Worker) String() string {
	return fmt.Sprintf("Anodot URL='%s'", w.metricsSubmitter.AnodotURL().Host)
}

var labels = []string{"anodot_url"}

var (
	concurrencyLimitReached = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_remote_write_concurrent_workers_limit_reached_total",
		Help: "Total count when concurrency limit was reached",
	}, labels)

	metricsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_remote_write_prometheus_samples_received_total",
		Help: "Total number of Prometheus metrics received",
	})

	concurrentWorkers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "anodot_remote_write_concurrent_workers",
		Help: "Total number of concurrent workers running currently",
	}, labels)

	bufferedMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "anodot_remote_write_buffered_metrics",
		Help: "Number of metrics stored in buffer awaiting buffer to be full",
	}, labels)

	bufferSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "anodot_remote_write_buffer_size",
		Help: "Anodot remote write metrics buffer size.",
	}, labels)

	anodotServerResponseTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "anodot_server_response_time_seconds",
		Help:       "Anodot server response time in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, labels)

	anodotSubmitterErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_metrics_submission_errors",
		Help: "Total number of errors occurred while sending metrics to Anodot api",
	}, labels)
)

func NewWorker(metricsSubmitter metrics.Submitter, workersLimit int64, debug bool) (*Worker, error) {
	if metricsSubmitter == nil {
		return nil, fmt.Errorf("metrics submitter should not be nil")
	}

	metricsPerRequestStr := os.Getenv("ANODOT_METRICS_PER_REQUEST_SIZE")
	if len(strings.TrimSpace(metricsPerRequestStr)) != 0 {
		v, e := strconv.Atoi(metricsPerRequestStr)
		if e == nil {
			metricsPerRequest = v
		}
	}

	log.Debug(fmt.Sprintf("Metrics per request size is : %d", metricsPerRequest))
	log.Debug(fmt.Sprintf("Metrics buffer size is : %d", maxMetricsBufferSize))

	worker := &Worker{metricsSubmitter: metricsSubmitter, Max: workersLimit, MetricsBuffer: make([]metrics.Anodot20Metric, 0, maxMetricsBufferSize), Current: 0, Debug: debug}
	return worker, nil
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(data []metrics.Anodot20Metric) {
	bufferSize.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(maxMetricsBufferSize))

	mutex.Lock()
	defer mutex.Unlock()

	metricsReceivedTotal.Add(float64(len(data)))
	if w.Debug {
		log.Debug(fmt.Sprintf("Received (%d) metrics: ", len(data)))
		for i := 0; i < len(data); i++ {
			log.Trace((data)[i])
		}
		return
	}

	w.MetricsBuffer = append(w.MetricsBuffer, data...)
	bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(len(w.MetricsBuffer)))

	log.Debug(fmt.Sprintf("Buffer size is %d", len(w.MetricsBuffer)))
	if len(w.MetricsBuffer) < metricsPerRequest {
		// need to wait until buffer will have enough elements to send
		return
	}

	metricsToSend := make([]metrics.Anodot20Metric, metricsPerRequest)

	copy(metricsToSend, w.MetricsBuffer[0:metricsPerRequest])
	w.MetricsBuffer = append(w.MetricsBuffer[:0], w.MetricsBuffer[metricsPerRequest:]...)
	bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(len(w.MetricsBuffer)))

	concurrentWorkers.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(w.Current))
	if w.Current >= w.Max {
		concurrencyLimitReached.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Inc()
		log.Warn("Reached workers concurrency limit. Sending metrics in single thread.")
		w.pushMetrics(w.metricsSubmitter, metricsToSend)
	} else {
		go func() {
			w.pushMetrics(w.metricsSubmitter, metricsToSend)
		}()
	}
}

func (w *Worker) pushMetrics(metricsSubmitter metrics.Submitter, metricsToSend []metrics.Anodot20Metric) {
	ts := time.Now()

	atomic.AddInt64(&w.Current, 1)
	defer atomic.AddInt64(&w.Current, -1)

	anodotResponse, err := metricsSubmitter.SubmitMetrics(metricsToSend)
	if err != nil {
		anodotSubmitterErrors.WithLabelValues(metricsSubmitter.AnodotURL().Host).Inc()
		log.Error("Failed to send metrics: ", err)
		return
	}

	if anodotResponse != nil && anodotResponse.HasErrors() {
		anodotSubmitterErrors.WithLabelValues(metricsSubmitter.AnodotURL().Host).Inc()
		log.Error("Anodot server returned error:", anodotResponse.ErrorMessage())
		return
	}

	anodotServerResponseTime.WithLabelValues(metricsSubmitter.AnodotURL().Host).Observe(time.Since(ts).Seconds())
}
