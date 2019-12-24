package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"io/ioutil"
	"log"
	"net/url"
	"sort"
	"testing"
	"time"
)

func TestMetricsShouldBeBuffered(t *testing.T) {
	expectedMetricsPerRequestSize := 1000

	mockSubmitter := &MockSubmitter{f: func(data []metrics.Anodot20Metric) (*metrics.AnodotResponse, error) {
		if len(data) != expectedMetricsPerRequestSize {
			t.Errorf(fmt.Sprintf("Submitted metreics size is %d. Required size is: %d", len(data), 1000))
		}

		if !sort.IsSorted(byTimestamp(data)) {
			t.Fatal("data should be sorted by time ASC")
		}
		return nil, nil
	}}

	worker, err := NewWorker(mockSubmitter, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	//nothing should be send here, only saved in buffer
	metrics := randomMetrics(2000)

	worker.Do(metrics[0:100])
	if len(worker.MetricsBuffer) != 100 {
		t.Fatal("metrics should be saved in buffer")
	}

	//nothing should be send here, only saved in buffer
	worker.Do(metrics[100:900])
	if len(worker.MetricsBuffer) != 900 {
		t.Fatal("metrics should be saved in buffer")
	}

	if !sort.IsSorted(byTimestamp(worker.MetricsBuffer)) {
		t.Fatal("metrics in buffer should be sorted by time ASC")
	}

	// 1000 metrics should be sent, so 100+800+600-1000=500
	worker.Do(metrics[900:1500])
	if len(worker.MetricsBuffer) != 500 {
		t.Fatal("metrics should be saved in buffer")
	}
	if !sort.IsSorted(byTimestamp(worker.MetricsBuffer)) {
		t.Fatal("metrics in buffer should be sorted by time ASC")
	}

	worker.Do(metrics[1500:2000])
	if len(worker.MetricsBuffer) != 0 {
		t.Fatal("metrics should be saved in buffer")
	}

}

type byTimestamp []metrics.Anodot20Metric

func (s byTimestamp) Len() int {
	return len(s)
}
func (s byTimestamp) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byTimestamp) Less(i, j int) bool {
	return s[i].Timestamp.UnixNano() <= s[j].Timestamp.UnixNano()
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
		m := metrics.Anodot20Metric{Value: float64(i),
			Timestamp: metrics.AnodotTimestamp{Time: time.Now().Add(time.Second * time.Duration(i))}}
		data = append(data, m)
	}

	return data
}
