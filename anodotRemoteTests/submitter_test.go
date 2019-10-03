package anodotRemoteTests

import (
	"encoding/json"
	"github.com/anodot/anodot-common/anodotParser"
	"testing"
)

//TODO vnekhai: move this to github.com/anodot/anodot-common ?

func TestSubmitter(t *testing.T) {
	excpedtedJonsOutput := `[{"properties":{"source":"gotest","target_type":"gauge","what":"test2"},"timestamp":1540153181,"value":1,"tags":{}},{"properties":{"source":"gotest","target_type":"gauge","what":"test2"},"timestamp":1540153181,"value":1,"tags":{}}]`

	metrics := make([]anodotParser.AnodotMetric, 0)
	metric := anodotParser.AnodotMetric{Properties: map[string]string{"what": "test2", "target_type": "gauge", "source": "gotest"}, Timestamp: 1540153181, Value: 1, Tags: map[string]string{}}
	metrics = append(metrics, metric)
	metrics = append(metrics, metric)

	mocksubmitter := MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
		b, e := json.Marshal(metrics)
		if e != nil {
			t.Fatal("Failed to convert metrics to json", e)
		}

		actualJson := string(b)
		if actualJson != excpedtedJonsOutput {
			t.Fatalf("expected metrics json: %v, \n got: %v", excpedtedJonsOutput, actualJson)
		}
	}}
	mocksubmitter.SubmitMetrics(metrics)
}

type MockSubmitter struct {
	f func(metrics []anodotParser.AnodotMetric)
}

func (s *MockSubmitter) SubmitMetrics(metrics []anodotParser.AnodotMetric) {
	s.f(metrics)
}
