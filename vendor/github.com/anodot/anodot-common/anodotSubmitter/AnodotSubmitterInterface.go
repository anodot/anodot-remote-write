package anodotSubmitter

import "github.com/anodot/anodot-common/anodotParser"

type Submitter interface {
	SubmitMetrics(metrics []anodotParser.AnodotMetric)
}
