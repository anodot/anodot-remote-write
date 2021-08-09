package prometheus

import (
	"github.com/anodot/anodot-common/pkg/metrics3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "k8s.io/klog/v2"
	"time"
)

func NewPipeline(startTime metrics3.AnodotTimestamp) metrics3.Pipeline {
	pipeline := metrics3.Pipeline{}
	pipeline.Id = "prometheus"
	pipeline.Created = startTime
	pipeline.Updated = startTime
	pipeline.Status = "RUNNING"

	s := metrics3.Source{}
	s.Name = "prometheus"
	s.Type = "prometheus"
	pipeline.Source = s

	return pipeline
}

var sendStatusToBCErrors = promauto.NewCounter(prometheus.CounterOpts{
	Name: "anodot_send_status_to_bc_errors",
	Help: "Total number of errors occurred while sending status to BC",
})

func SendAgentStatusToBC(client *metrics3.Anodot30Client, sendToBCPeriod int) {
	startTime := metrics3.AnodotTimestamp{Time: time.Now()}
	go func() {
		ticker := time.NewTicker(time.Duration(sendToBCPeriod) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			_, err := client.SendToBC(NewPipeline(startTime))
			if err != nil {
				sendStatusToBCErrors.Add(1)
				log.Errorf("Failed to send status to BC %v", err)
			}
		}
	}()
}
