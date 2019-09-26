package anodotServer

import (
	"fmt"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"github.com/anodot/anodot-common/remoteStats"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const BUFFER_SIZE = 1000

type Worker struct {
	Max           int64
	Current       int64
	Stats         *remoteStats.RemoteStats
	MetricsBuffer []anodotParser.AnodotMetric
	Debug         bool
}

func NewWorker(max int64, stats *remoteStats.RemoteStats, debug bool) Worker {
	return Worker{Max: max, MetricsBuffer: make([]anodotParser.AnodotMetric, 0), Current: 0, Stats: stats, Debug: debug}
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(metrics *[]anodotParser.AnodotMetric,
	s *anodotSubmitter.Anodot20Submitter) {

	mutex.Lock()
	defer mutex.Unlock()

	if len(w.MetricsBuffer) < BUFFER_SIZE {
		w.MetricsBuffer = append(w.MetricsBuffer, *metrics...)
		return
	}

	newArray := make([]anodotParser.AnodotMetric, len(w.MetricsBuffer))
	copy(newArray, w.MetricsBuffer)
	w.MetricsBuffer = make([]anodotParser.AnodotMetric, 0)

	if w.Debug == true {
		for i := 0; i < len(newArray); i++ {
			fmt.Println((newArray)[i])
		}
		return
	}

	if w.Current >= w.Max {
		log.Println("Moving to a Blocking Mode !!!")
		s.SubmitMetrics(&newArray)
	} else {
		go func() {
			ts := time.Now()
			w.Stats.UpdateGauge(remoteStats.CONCURRENT_REQUESTS, w.Current)
			atomic.AddInt64(&w.Current, 1)
			defer atomic.AddInt64(&w.Current, -1)
			s.SubmitMetrics(&newArray)
			if s.MirrorUrl != "" {
				s.MirrorMetrics(&newArray)
			}
			w.Stats.UpdateHist(remoteStats.REMOTE_REQUEST_TIME, int64(time.Since(ts).Seconds()))
			return
		}()
	}
}
