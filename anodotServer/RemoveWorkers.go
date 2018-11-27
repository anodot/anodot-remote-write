package anodotServer

import (
	"github.com/anodot/anodot-common/anodotSubmitter"
	"github.com/anodot/anodot-common/anodotParser"
	"sync/atomic"
	"../remoteStats"
	"sync"
	"time"
	"log"
)

const BUFFER_SIZE  = 1000

type Worker struct {
	Max  int64
	Current int64
	Stats *remoteStats.RemoteStats
	MetricsBuffer []anodotParser.AnodotMetric
}

func NewWorker(max int64,stats *remoteStats.RemoteStats)  Worker{
	return  Worker{Max:max,MetricsBuffer: make([]anodotParser.AnodotMetric,0),Current:0,Stats:stats}
}

var mutex = &sync.Mutex{}

func (w *Worker) Do(metrics *[]anodotParser.AnodotMetric,
	s* anodotSubmitter.Anodot20Submitter) {

		mutex.Lock()
		defer mutex.Unlock()

		if len(w.MetricsBuffer) < BUFFER_SIZE {
			w.MetricsBuffer = append(w.MetricsBuffer, *metrics...)
			return
		}

		newArray  := make([]anodotParser.AnodotMetric, len(w.MetricsBuffer))
		copy(newArray,w.MetricsBuffer)
		w.MetricsBuffer = make([]anodotParser.AnodotMetric,0)

		if w.Current >= w.Max {
			log.Println("Moving to a Blocking Mode !!!")
			s.SubmitMetrics(&newArray)
		} else {
			go func() {
				ts := time.Now()
				w.Stats.UpdateGauge(remoteStats.CONCURRENT_REQUESTS,w.Current)
				atomic.AddInt64(&w.Current, 1)
				defer atomic.AddInt64(&w.Current,-1)
				s.SubmitMetrics(&newArray)
				w.Stats.UpdateHist(remoteStats.REMOTE_REQUEST_TIME,int64(time.Since(ts).Seconds()))
				return
			}()
		}
}
