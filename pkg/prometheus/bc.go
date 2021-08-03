package prometheus

import (
	"github.com/anodot/anodot-common/pkg/metrics3"
	"time"
)

func NewPipeline(startTime time.Time) metrics3.Pipeline {
	pipeline := metrics3.Pipeline{}
	pipeline.Id = "prometheus"
	startTimeStr := startTime.Unix()
	pipeline.Created = startTimeStr
	pipeline.Updated = startTimeStr
	pipeline.Status = "RUNNING"

	s := metrics3.Source{}
	s.Name = "prometheus"
	s.Type = "prometheus"
	pipeline.Source = s

	return pipeline
}
