package remote

import (
	"encoding/json"
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/kelseyhightower/envconfig"
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

	currentWorkers int64

	mu            sync.RWMutex
	MetricsBuffer []metrics.Anodot20Metric
	flushBuffer   chan bool

	*WorkerConfig
}

type WorkerConfig struct {
	BatchSendDeadline time.Duration `default:"1m" split_words:"true"`
	MaxAllowedEPS     int           `default:"0" split_words:"true"`

	MaxWorkers            int64 `default:"20" split_words:"true" `
	MetricsPerRequestSize int   `default:"1000" split_words:"true"`

	Debug bool `default:"false"`
}

func NewWorkerConfig() (*WorkerConfig, error) {
	config := &WorkerConfig{}
	err := envconfig.Process("ANODOT", config)

	if config.MaxWorkers < 0 {
		config.MaxWorkers = 20
	}

	if config.MetricsPerRequestSize <= 0 {
		config.MetricsPerRequestSize = 1000
	}

	if config.MaxAllowedEPS != 0 && config.MaxAllowedEPS < config.MetricsPerRequestSize {
		return nil, fmt.Errorf("ANODOT_MAX_ALLOWED_EPS should be grather than ANODOT_METRICS_PER_REQUEST_SIZE")
	}

	return config, err
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

func (w *Worker) LastTimestamp() *time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.MetricsBuffer) == 0 {
		return nil
	}

	res := w.MetricsBuffer[len(w.MetricsBuffer)-1].Timestamp
	return &res.Time
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

	throttlingTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "anodot_workers_throttle_time_ms",
		Help: "Anodot workers time throttled",
	}, labels)
)

func NewWorker(metricsSubmitter metrics.Submitter, config *WorkerConfig) (*Worker, error) {
	if metricsSubmitter == nil {
		return nil, fmt.Errorf("metrics submitter should not be nil")
	}

	if config == nil {
		return nil, fmt.Errorf("worker config should not be nil")
	}

	maxEPSString := os.Getenv("ANODOT_MAX_ALLOWED_EPS")
	maxAllowedEps := 0
	if len(strings.TrimSpace(maxEPSString)) > 0 {
		var err error
		maxAllowedEps, err = strconv.Atoi(maxEPSString)
		if err != nil {
			return nil, err
		}
	}

	worker := &Worker{metricsSubmitter: metricsSubmitter, WorkerConfig: config, MetricsBuffer: make([]metrics.Anodot20Metric, 0, 100000), flushBuffer: make(chan bool, 50)}

	log.V(4).Infof("Metrics per request size is : %d", worker.MetricsPerRequestSize)
	log.V(4).Infof("Metrics buffer size is : %d", len(worker.MetricsBuffer))

	bufferSize.WithLabelValues(worker.metricsSubmitter.AnodotURL().Host).Set(float64(len(worker.MetricsBuffer)))

	var throttle *time.Ticker
	if worker.MaxAllowedEPS > 0 {
		duration := time.Duration(1e6/(maxAllowedEps/worker.MetricsPerRequestSize)) * time.Microsecond
		log.V(2).Infof("Throttling interval is: %s", duration.String())
		throttle = time.NewTicker(duration)
	}

	//used to clean metrics buffer by expiration time
	go func(w *Worker) {
		ticker := time.NewTicker(w.BatchSendDeadline)
		defer ticker.Stop()
		for range ticker.C {
			timestamp := w.LastTimestamp()
			if timestamp == nil {
				continue
			}
			if time.Since(*timestamp) > w.BatchSendDeadline {
				log.V(4).Infof("reached BatchSendDeadline of '%s'. Flushing metrics buffer", w.BatchSendDeadline.String())
				w.flushBuffer <- true
			}
		}
	}(worker)

	go func(w *Worker) {
		for {
			<-w.flushBuffer
			bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(w.BufferSize()))

			var chunkSize int

			for w.BufferSize() > 0 {
				//throttle if needed
				if w.MaxAllowedEPS > 0 {
					start := time.Now()
					<-throttle.C
					throttlingTime.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Add(float64(time.Since(start).Milliseconds()))
				}

				w.mu.Lock()

				if len(w.MetricsBuffer) > w.MetricsPerRequestSize {
					chunkSize = w.MetricsPerRequestSize
				} else {
					chunkSize = len(w.MetricsBuffer)
				}

				metricsToSend := make([]metrics.Anodot20Metric, chunkSize)
				copy(metricsToSend, w.MetricsBuffer[0:chunkSize])
				w.MetricsBuffer = append(w.MetricsBuffer[:0], w.MetricsBuffer[chunkSize:]...)
				w.mu.Unlock()

				if atomic.LoadInt64(&w.currentWorkers) >= w.MaxWorkers {
					concurrencyLimitReached.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Inc()
					log.Warning("Reached workers concurrency limit. Sending metrics in single thread.")
					w.pushMetrics(w.metricsSubmitter, metricsToSend)
				} else {
					go func() {
						w.pushMetrics(w.metricsSubmitter, metricsToSend)
					}()
				}
			}
			bufferedMetrics.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(w.BufferSize()))
			concurrentWorkers.WithLabelValues(w.metricsSubmitter.AnodotURL().Host).Set(float64(atomic.LoadInt64(&w.currentWorkers)))
		}
	}(worker)

	return worker, nil
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(data []metrics.Anodot20Metric) {
	//TODO check if we need this mutex
	mutex.Lock()
	defer mutex.Unlock()

	log.V(4).Infof("Received (%d) metric(s): ", len(data))
	metricsReceivedTotal.Add(float64(len(data)))
	if w.Debug {
		bytes, err := json.Marshal(data)
		if err != nil {
			log.Error("failed to display metrics:", err)
		}
		log.V(5).Info(string(bytes))
		return
	}

	w.mu.Lock()
	w.MetricsBuffer = append(w.MetricsBuffer, data...)
	if len(w.MetricsBuffer) >= w.MetricsPerRequestSize {
		w.flushBuffer <- true
	}
	w.mu.Unlock()
}

func (w *Worker) pushMetrics(metricsSubmitter metrics.Submitter, metricsToSend []metrics.Anodot20Metric) {
	ts := time.Now()

	atomic.AddInt64(&w.currentWorkers, 1)
	defer atomic.AddInt64(&w.currentWorkers, -1)

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
