package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var metricsPerRequest = 1000
var maxMetricsBufferSize = 1000

type Worker struct {
	Max           int64
	Current       int64
	MetricsBuffer []metrics.Anodot20Metric
	Debug         bool
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

func NewWorker(workersLimit int64, debug bool) *Worker {
	metricsPerRequestStr := os.Getenv("ANODOT_METRICS_BUFFER_SIZE")
	if len(strings.TrimSpace(metricsPerRequestStr)) != 0 {
		v, e := strconv.Atoi(metricsPerRequestStr)
		if e == nil {
			maxMetricsBufferSize = v
		}
	}

	log.Println(fmt.Sprintf("Metrics per request size is : %d", metricsPerRequest))
	log.Println(fmt.Sprintf("Metrics buffer size is : %d", maxMetricsBufferSize))
	return &Worker{Max: workersLimit, MetricsBuffer: make([]metrics.Anodot20Metric, 0, maxMetricsBufferSize), Current: 0, Debug: debug}
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(data []metrics.Anodot20Metric, s metrics.Submitter) {
	bufferSize.WithLabelValues(s.AnodotURL().Host).Set(float64(maxMetricsBufferSize))

	mutex.Lock()
	defer mutex.Unlock()

	metricsReceivedTotal.Add(float64(len(data)))
	if w.Debug {
		log.Println("Received metrics: ")
		for i := 0; i < len(data); i++ {
			log.Println((data)[i])
		}
		return
	}

	w.MetricsBuffer = append(w.MetricsBuffer, data...)
	bufferedMetrics.WithLabelValues(s.AnodotURL().Host).Set(float64(len(w.MetricsBuffer)))
	if len(w.MetricsBuffer) < maxMetricsBufferSize {
		// need to wait until buffer will have enough elements to send
		return
	}

	metricsToSend := make([]metrics.Anodot20Metric, len(w.MetricsBuffer))
	copy(metricsToSend, w.MetricsBuffer)
	w.MetricsBuffer = nil
	bufferedMetrics.WithLabelValues(s.AnodotURL().Host).Set(float64(len(w.MetricsBuffer)))

	//send metrics by chunks with size = metricsPerRequest
	for i := 0; i < len(metricsToSend); i += metricsPerRequest {
		end := i + metricsPerRequest

		if end > len(metricsToSend) {
			end = len(metricsToSend)
		}

		anodotMetricsChunk := metricsToSend[i:end]
		concurrentWorkers.WithLabelValues(s.AnodotURL().Host).Set(float64(w.Current))
		if w.Current >= w.Max {
			concurrencyLimitReached.WithLabelValues(s.AnodotURL().Host).Inc()
			log.Println("Reached workers concurrency limit. Sending metrics in single thread.")
			w.pushMetrics(s, anodotMetricsChunk)
		} else {
			go func() {
				w.pushMetrics(s, anodotMetricsChunk)
			}()
		}
	}
}

func (w *Worker) pushMetrics(metricsSubmitter metrics.Submitter, metricsToSend []metrics.Anodot20Metric) {
	ts := time.Now()

	atomic.AddInt64(&w.Current, 1)
	defer atomic.AddInt64(&w.Current, -1)

	anodotResponse, err := metricsSubmitter.SubmitMetrics(metricsToSend)
	if err != nil {
		anodotSubmitterErrors.WithLabelValues(metricsSubmitter.AnodotURL().Host).Inc()
		log.Println("[ERROR] failed to send metrics: ", err)
		return
	}

	if anodotResponse != nil && anodotResponse.HasErrors() {
		anodotSubmitterErrors.WithLabelValues(metricsSubmitter.AnodotURL().Host).Inc()
		log.Println("[ERROR] anodot api error : ", err)
		return
	}

	anodotServerResponseTime.WithLabelValues(metricsSubmitter.AnodotURL().Host).Observe(time.Since(ts).Seconds())
}
