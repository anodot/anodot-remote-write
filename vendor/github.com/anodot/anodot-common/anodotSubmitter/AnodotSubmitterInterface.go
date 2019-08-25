package anodotSubmitter

import "github.com/anodot/anodot-common/anodotParser"

type submitter interface {
	SubmitMetrics(metrics *[]anodotParser.AnodotMetric)()
}
