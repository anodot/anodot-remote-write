package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"io/ioutil"
	"log"
	"net/url"
	"testing"
)

//TODO add test for publishing metrics
func TestMetricsSizeAboveBuffer(t *testing.T) {
	var anodotRequestNumber int

	reqSize := 1500

	mockSubmitter := &MockSubmitter{f: func(data []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
		if anodotRequestNumber == 0 && len(data) != 1000 {
			t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(data), 1000))
		}
		// on second request  - remaining data
		if anodotRequestNumber == 1 && len(data) != 500 {
			t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(data), 200))
		}
		anodotRequestNumber++
		return nil, nil
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
		{200, MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
			t.Fatal("1 Should not send any metrics if buffer not full")
			return nil, nil
		}}},
		{300, MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
			t.Fatal("2 Should not send any metrics if buffer not full")
			return nil, nil
		}}},
		{700, MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
			// on first request we need to send exactly 1000 metrics
			if anodotRequestNumber == 0 && len(metrics) != 1000 {
				t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 1000))
			}
			// on second request  - remaining metrics
			if anodotRequestNumber == 1 && len(metrics) != 200 {
				t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(metrics), 200))
			}
			anodotRequestNumber++

			return nil, nil
		}}},
	}

	worker := NewWorker(0, false)

	for _, data := range testData {
		worker.Do(randomMetrics(data.metricsSize), &data.MockSubmitter)
	}
}

func TestNoMetricsSendInDebugMode(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	reqSize := 1500
	mockSubmitter := &MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
		t.Errorf("No metrics should be sent in debug mode")
		return nil, nil
	}}

	worker := NewWorker(0, true)
	worker.Do(randomMetrics(reqSize), mockSubmitter)
}

type MockSubmitter struct {
	f func([]metrics.Anodot20Metric) (*metrics.AnodotResponse, error)
}

func (m MockSubmitter) SubmitMetrics(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
	return m.f(metrics)
}

func (m MockSubmitter) AnodotURL() *url.URL {
	return nil
}

func randomMetrics(size int) []metrics.Anodot20Metric {
	data := make([]metrics.Anodot20Metric, 0, size)
	for i := 0; i < size; i++ {
		m := metrics.Anodot20Metric{Value: float64(i)}
		data = append(data, m)
	}

	return data
}
