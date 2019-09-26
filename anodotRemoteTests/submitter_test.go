package anodotRemoteTests

import (
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"testing"
)

func TestSubmitter(t *testing.T) {
	metrics := make([]anodotParser.AnodotMetric, 0)
	metric := anodotParser.AnodotMetric{Properties: map[string]string{"what": "test2", "target_type": "gauge", "source": "gotest"}, Timestamp: 1540153181, Value: 1, Tags: map[string]string{}}
	metrics = append(metrics, metric)
	metrics = append(metrics, metric)
	mocksubmitter := anodotSubmitter.MockSubmitter{}

	if mocksubmitter.SubmitMetrics(&metrics) != nil {
		t.Fail()
	}
}
