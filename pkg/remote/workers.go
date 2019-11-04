package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
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
	MetricsBuffer []anodotParser.AnodotMetric
	Debug         bool
}

var (
	concurrencyLimitReached = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_remote_write_concurrent_workers_limit_reached_total",
		Help: "Total count when concurrency limit was reached",
	})

	metricsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_remote_write_prometheus_samples_received_total",
		Help: "Total number of Prometheus metrics received",
	})

	concurrentWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "anodot_remote_write_concurrent_workers",
		Help: "Total number of concurrent workers running currently",
	})

	bufferedMetrics = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "anodot_remote_write_buffered_metrics",
		Help: "Number of metrics stored in buffer awaiting buffer to be full",
	})

	bufferSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "anodot_remote_write_buffer_size",
		Help: "Anodot remote write metrics buffer size.",
	})

	anodotServerResponseTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "anodot_server_response_time_seconds",
		Help:       "Anodot server response time in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
)

func NewWorker(workersLimit int64, debug bool) Worker {
	metricsPerRequestStr := os.Getenv("ANODOT_METRICS_BUFFER_SIZE")
	if len(strings.TrimSpace(metricsPerRequestStr)) != 0 {
		v, e := strconv.Atoi(metricsPerRequestStr)
		if e == nil {
			maxMetricsBufferSize = v
		}
	}

	bufferSize.Set(float64(maxMetricsBufferSize))
	log.Println(fmt.Sprintf("Metrics per request size is : %d", metricsPerRequest))
	log.Println(fmt.Sprintf("Metrics buffer size is : %d", maxMetricsBufferSize))
	return Worker{Max: workersLimit, MetricsBuffer: make([]anodotParser.AnodotMetric, 0, maxMetricsBufferSize), Current: 0, Debug: debug}
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(metrics []anodotParser.AnodotMetric, s anodotSubmitter.Submitter) {

	mutex.Lock()
	defer mutex.Unlock()

	metricsReceivedTotal.Add(float64(len(metrics)))
	if w.Debug {
		log.Println("Received metrics: ")
		for i := 0; i < len(metrics); i++ {
			log.Println((metrics)[i])
		}
		return
	}

	w.MetricsBuffer = append(w.MetricsBuffer, metrics...)
	bufferedMetrics.Set(float64(len(w.MetricsBuffer)))
	if len(w.MetricsBuffer) < maxMetricsBufferSize {
		// need to wait until buffer will have enough elements to send
		return
	}

	metricsToSend := make([]anodotParser.AnodotMetric, len(w.MetricsBuffer))
	copy(metricsToSend, w.MetricsBuffer)
	w.MetricsBuffer = nil
	bufferedMetrics.Set(float64(len(w.MetricsBuffer)))

	//send metrics by chunks with size = metricsPerRequest
	for i := 0; i < len(metricsToSend); i += metricsPerRequest {
		end := i + metricsPerRequest

		if end > len(metricsToSend) {
			end = len(metricsToSend)
		}

		anodotMetricsChunk := metricsToSend[i:end]
		concurrentWorkers.Set(float64(w.Current))
		if w.Current >= w.Max {
			concurrencyLimitReached.Inc()
			log.Println("Reached workers concurrency limit. Sending metrics in single thread.")
			w.pushMetrics(s, anodotMetricsChunk)
		} else {
			go func() {
				w.pushMetrics(s, anodotMetricsChunk)
			}()
		}
	}
}

func (w *Worker) pushMetrics(metricsSubmitter anodotSubmitter.Submitter, metricsToSend []anodotParser.AnodotMetric) {
	ts := time.Now()

	atomic.AddInt64(&w.Current, 1)
	defer atomic.AddInt64(&w.Current, -1)

	metricsSubmitter.SubmitMetrics(metricsToSend)

	//TODO: remote MirrorMetrics or add MirrorMetrics to interface
	s, ok := metricsSubmitter.(*anodotSubmitter.Anodot20Submitter)
	if ok {
		if s.MirrorUrl != "" {
			s.MirrorMetrics(metricsToSend)
		}
	}
	anodotServerResponseTime.Observe(time.Since(ts).Seconds())
}
