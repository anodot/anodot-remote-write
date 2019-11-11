package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/anodotParser"
	"io/ioutil"
	"log"
	"testing"
)

//TODO add test for publishing metrics
func TestMetricsSizeAboveBuffer(t *testing.T) {
	var anodotRequestNumber int

	reqSize := 1500

	mockSubmitter := &MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
		if anodotRequestNumber == 0 && len(metrics) != 1000 {
			t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 1000))
		}
		// on second request  - remaining metrics
		if anodotRequestNumber == 1 && len(metrics) != 500 {
			t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 200))
		}
		anodotRequestNumber++
	}}

	worker := NewWorker(0, false)
	worker.Do(randomMetrics(reqSize), mockSubmitter)
}

func TestMetricsShouldBeBuffered(t *testing.T) {
	var anodotRequestNumber int

	var testData = []struct {
		metricsSize   int
		MockSubmitter MockSubmitter
	}{
		{200, MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
			t.Fatal("1 Should not send any metrics if buffer not full")
		}}},
		{300, MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
			t.Fatal("2 Should not send any metrics if buffer not full")
		}}},
		{700, MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
			// on first request we need to send exactly 1000 metrics
			if anodotRequestNumber == 0 && len(metrics) != 1000 {
				t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 1000))
			}
			// on second request  - remaining metrics
			if anodotRequestNumber == 1 && len(metrics) != 200 {
				t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 200))
			}
			anodotRequestNumber++
		}}},
	}

	worker := NewWorker(0, false)

	for _, data := range testData {
		worker.Do(randomMetrics(data.metricsSize), data.MockSubmitter)
	}
}

func TestNoMetricsSendInDebugMode(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	reqSize := 1500
	mockSubmitter := &MockSubmitter{f: func(metrics []anodotParser.AnodotMetric) {
		t.Errorf("No metrics should be sent in debug mode")
	}}

	worker := NewWorker(0, true)
	worker.Do(randomMetrics(reqSize), mockSubmitter)
}

type MockSubmitter struct {
	f func([]anodotParser.AnodotMetric)
}

func (m MockSubmitter) SubmitMetrics(metrics []anodotParser.AnodotMetric) {
	m.f(metrics)
}

func randomMetrics(size int) []anodotParser.AnodotMetric {
	metrics := make([]anodotParser.AnodotMetric, 0, size)
	for i := 0; i < size; i++ {
		m := anodotParser.AnodotMetric{Value: float64(i)}
		metrics = append(metrics, m)
	}

	return metrics
}
