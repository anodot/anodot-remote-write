package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "k8s.io/klog/v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Worker struct {
	metricsSubmitter metrics.Submitter

	metricsPerRequest int

	Max     int64
	Current int64

	mu            sync.RWMutex
	MetricsBuffer []metrics.Anodot20Metric
	flushBuffer   chan bool
	Debug         bool
}

func (w *Worker) String() string {
	return fmt.Sprintf("Anodot URL='%s'", w.metricsSubmitter.AnodotURL().Host)
}

func (w *Worker) BufferSize() int {
	w.mu.RLock()
	size := len(w.MetricsBuffer)
	w.mu.RUnlock()

	return size
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

	serverHTTPResponses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_server_http_responses_total",
		Help: "Total number of HTTP responses of Anodot server",
	}, []string{"anodot_url", "response_code"})
)

func NewWorker(metricsSubmitter metrics.Submitter, workersLimit int64, debug bool) (*Worker, error) {
	if metricsSubmitter == nil {
		return nil, fmt.Errorf("metrics submitter should not be nil")
	}

	if workersLimit < 0 {
		return nil, fmt.Errorf("workersLimit should be > 0")
	}

	worker := &Worker{metricsSubmitter: metricsSubmitter, Max: workersLimit, MetricsBuffer: make([]metrics.Anodot20Metric, 0, 100000), Debug: debug, flushBuffer: make(chan bool)}

	metricsPerRequestStr := os.Getenv("ANODOT_METRICS_PER_REQUEST_SIZE")
	if len(strings.TrimSpace(metricsPerRequestStr)) != 0 {
		v, e := strconv.Atoi(metricsPerRequestStr)
		if e == nil {
			worker.metricsPerRequest = v
		}
	}

	if worker.metricsPerRequest <= 0 {
		worker.metricsPerRequest = 1000
	}

	log.V(4).Infof("Metrics per request size is : %d", worker.metricsPerRequest)
	log.V(4).Infof("Metrics buffer size is : %d", len(worker.MetricsBuffer))

	bufferSize.WithLabelValues(worker.metricsSubmitter.AnodotURL().Host).Set(float64(len(worker.MetricsBuffer)))

	go func(w *Worker) {
		for {
			<-w.flushBuffer
			buffSize := w.BufferSize()

			bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(buffSize))
			for w.BufferSize() >= w.metricsPerRequest {
				metricsToSend := make([]metrics.Anodot20Metric, w.metricsPerRequest)

				w.mu.Lock()
				copy(metricsToSend, w.MetricsBuffer[0:w.metricsPerRequest])
				w.MetricsBuffer = append(w.MetricsBuffer[:0], w.MetricsBuffer[w.metricsPerRequest:]...)
				w.mu.Unlock()

				if w.Current >= w.Max {
					concurrencyLimitReached.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Inc()
					log.Warning("Reached workers concurrency limit. Sending metrics in single thread.")
					w.pushMetrics(w.metricsSubmitter, metricsToSend)
				} else {
					go func() {
						w.pushMetrics(w.metricsSubmitter, metricsToSend)
					}()
				}
			}

			bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(buffSize))
			concurrentWorkers.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(atomic.LoadInt64(&w.Current)))
		}
	}(worker)

	return worker, nil
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(data []metrics.Anodot20Metric) {
	//TODO check if we need this mutex
	mutex.Lock()
	defer mutex.Unlock()

	log.V(4).Infof("Received (%d) metrics: ", len(data))
	metricsReceivedTotal.Add(float64(len(data)))
	if w.Debug {
		for i := 0; i < len(data); i++ {
			log.V(5).Info((data)[i])
		}
		return
	}

	w.mu.Lock()
	w.MetricsBuffer = append(w.MetricsBuffer, data...)
	w.mu.Unlock()

	if w.BufferSize() > w.metricsPerRequest {
		w.flushBuffer <- true
	}
}

func (w *Worker) pushMetrics(metricsSubmitter metrics.Submitter, metricsToSend []metrics.Anodot20Metric) {
	ts := time.Now()

	atomic.AddInt64(&w.Current, 1)
	defer atomic.AddInt64(&w.Current, -1)

	anodotResponse, err := metricsSubmitter.SubmitMetrics(metricsToSend)
	if anodotResponse != nil && anodotResponse.RawResponse() != nil {
		serverHTTPResponses.WithLabelValues(w.metricsSubmitter.AnodotURL().Host, strconv.Itoa(anodotResponse.RawResponse().StatusCode)).Inc()
	}
	if err != nil {
		anodotSubmitterErrors.WithLabelValues(metricsSubmitter.AnodotURL().Host).Inc()
		log.Error("Failed to send metrics: ", err)
		return
	}

	anodotServerResponseTime.WithLabelValues(metricsSubmitter.AnodotURL().Host).Observe(time.Since(ts).Seconds())
}
