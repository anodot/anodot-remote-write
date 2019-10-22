package remote

import (
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"github.com/anodot/anodot-common/remoteStats"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const metricsPerRequest = 1000

type Worker struct {
	Max           int64
	Current       int64
	Stats         *remoteStats.RemoteStats
	MetricsBuffer []anodotParser.AnodotMetric
	Debug         bool
}

func NewWorker(workersLimit int64, stats *remoteStats.RemoteStats, debug bool) Worker {
	return Worker{Max: workersLimit, MetricsBuffer: make([]anodotParser.AnodotMetric, 0), Current: 0, Stats: stats, Debug: debug}
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(metrics []anodotParser.AnodotMetric, s anodotSubmitter.Submitter) {

	mutex.Lock()
	defer mutex.Unlock()

	if w.Debug {
		log.Println("Received metrics:")
		for i := 0; i < len(metrics); i++ {
			log.Println((metrics)[i])
		}
		return
	}

	w.MetricsBuffer = append(w.MetricsBuffer, metrics...)
	if len(w.MetricsBuffer) < metricsPerRequest {
		// need to wait until buffer will have enough elements to send
		return
	}

	metricsToSend := make([]anodotParser.AnodotMetric, len(w.MetricsBuffer))
	copy(metricsToSend, w.MetricsBuffer)
	w.MetricsBuffer = nil

	//send metrics by chunks with size = metricsPerRequest
	for i := 0; i < len(metricsToSend); i += metricsPerRequest {
		end := i + metricsPerRequest

		if end > len(metricsToSend) {
			end = len(metricsToSend)
		}

		anodotMetricsChunk := metricsToSend[i:end]
		if w.Current >= w.Max {
			//TODO vnekhai: add counter when concurrency limit reached.
			log.Println("Reached workers concurrency limit. Sending metrics in single thread.")
			w.pushMetrics(s, anodotMetricsChunk)
		} else {
			go func() {
				w.Stats.UpdateGauge(remoteStats.CONCURRENT_REQUESTS, w.Current)
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

	w.Stats.UpdateHist(remoteStats.REMOTE_REQUEST_TIME, int64(time.Since(ts).Seconds()))
}
