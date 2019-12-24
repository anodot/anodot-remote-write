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

	worker, err := NewWorker(mockSubmitter, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	worker.Do(randomMetrics(reqSize))
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

	for _, data := range testData {
		worker, err := NewWorker(data.MockSubmitter, 0, false)
		if err != nil {
			t.Fatal(err)
		}
		worker.Do(randomMetrics(data.metricsSize))
	}
}

func TestNoMetricsSendInDebugMode(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	reqSize := 1500
	mockSubmitter := &MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
		t.Errorf("No metrics should be sent in debug mode")
		return nil, nil
	}}

	worker, err := NewWorker(mockSubmitter, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	worker.Do(randomMetrics(reqSize))
}

type MockSubmitter struct {
	f func([]metrics.Anodot20Metric) (*metrics.AnodotResponse, error)
}

func (m MockSubmitter) SubmitMetrics(metrics []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
	return m.f(metrics)
}

func (m MockSubmitter) AnodotURL() *url.URL {
	parse, _ := url.Parse("http://127.0.0.1")
	return parse
}

func randomMetrics(size int) []metrics.Anodot20Metric {
	data := make([]metrics.Anodot20Metric, 0, size)
	for i := 0; i < size; i++ {
		m := metrics.Anodot20Metric{Value: float64(i)}
		data = append(data, m)
	}

	return data
}

func TestSlice(t *testing.T) {
	a := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	b := make([]string, 0, 5)

	fmt.Println(b)
	fmt.Println(len(b))
	fmt.Println(cap(b))

	copy(b, a[0:5])
	a = append(a[:0], a[5:]...)
	a = append(a, "11")
	a = append(a, "12")

	fmt.Println(a)
	fmt.Println(len(a))
	fmt.Println(cap(a))

	fmt.Println(b)
	fmt.Println(len(b))
	fmt.Println(cap(b))

}
