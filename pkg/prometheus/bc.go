package prometheus

import (
	"github.com/anodot/anodot-common/pkg/metrics3/bc"
	"time"
)

func NewPipeline(startTime time.Time) Pipeline {
	pipeline := bc.Pipeline{}
	pipeline.Id = "prometheus"
	startTimeStr := time.Date(startTime)
	pipeline.Created = startTimeStr
	pipeline.Updated = startTimeStr
	pipeline.Status = "RUNNING"
	s := SourceType{}
	s.Name = "prometheus"
	s.Type = "prometheus"
	pipeline.Source = s
}
