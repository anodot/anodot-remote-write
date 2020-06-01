package remote

import (
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"net/http"
	"strings"
	"sync/atomic"

	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestMetricsShouldBeBuffered(t *testing.T) {
	expectedMetricsPerRequestSize := map[int]int{1: 1000, 2: 500}

	requestNumber := 0
	mockSubmitter := &MockSubmitter{f: func(data []metrics.Anodot20Metric) (metrics.AnodotResponse, error) {
		requestNumber++
		expectedMetricsSize := expectedMetricsPerRequestSize[requestNumber]
		if len(data) != expectedMetricsSize {
			t.Errorf(fmt.Sprintf("Submitted metrics size is %d. Required size is: %d", len(data), expectedMetricsSize))
		}

		if !sort.IsSorted(byTimestamp(data)) {
			t.Fatal("data should be sorted by time ASC")
		}
		return nil, nil
	}}

	err := os.Setenv("ANODOT_MAX_WORKERS", "0")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer func() {
		err := os.Unsetenv("ANODOT_MAX_WORKERS")
		if err != nil {
			t.Fatalf(err.Error())
		}
	}()

	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}

	worker, err := NewWorker(mockSubmitter, config)
	if err != nil {
		t.Fatal(err)
	}

	allMetrics := randomMetrics(2000)

	//nothing should be send here, only saved in buffer
	worker.Do(allMetrics[0:100])
	if len(worker.MetricsBuffer) != 100 {
		t.Fatal("metrics should be saved in buffer")
	}

	//nothing should be send here, only saved in buffer
	worker.Do(allMetrics[100:900])
	if len(worker.MetricsBuffer) != 900 {
		t.Fatal("metrics should be saved in buffer")
	}

	if !sort.IsSorted(byTimestamp(worker.MetricsBuffer)) {
		t.Fatal("metrics in buffer should be sorted by time ASC")
	}

	if !sort.IsSorted(byTimestamp(worker.MetricsBuffer)) {
		t.Fatal("metrics in buffer should be sorted by time ASC")
	}

	// all metrics which were accumulated before - should be sent, once buffer has reached capacity
	worker.Do(allMetrics[900:1500])
	//fist batch
	waitWorkers(worker, 1)
	waitWorkers(worker, 0)
	//second batch
	waitWorkers(worker, 1)
	waitWorkers(worker, 0)
	if len(worker.MetricsBuffer) != 0 {
		t.Fatalf("empty buffer expcted")
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

func TestToString(t *testing.T) {
	anodot20Submitter, e := metrics.NewAnodot20Client("http://localhost:8080", "123", nil)
	if e != nil {
		t.Fatal(e)
	}
	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}
	worker, e := NewWorker(anodot20Submitter, config)
	if e != nil {
		t.Fatal(e)
	}

	expectedRes := "Anodot URL='localhost:8080'"
	if worker.String() != expectedRes {
		t.Fatal(fmt.Sprintf("Wrong String() result\n got: %q\n want: %q", worker.String(), expectedRes))
	}
}

func TestMetricsPerRequestEnvConfigurationIncorrect(t *testing.T) {

	var envConfig = []struct {
		envValue string
		isValid  bool
	}{
		{"", false},
		{" ", false},
		{"asd", false},
		{"-10", false},
		{"500", true},
	}

	for _, v := range envConfig {
		t.Run(fmt.Sprintf("ANODOT_METRICS_PER_REQUEST_SIZE is %q", v.envValue),
			func(t *testing.T) {

				err := os.Setenv("ANODOT_METRICS_PER_REQUEST_SIZE", v.envValue)
				if err != nil {
					t.Fatal(err)
				}

				config, err := NewWorkerConfig()
				if err != nil {
					if v.isValid {
						t.Fatal(err)
					}
				}

				worker, e := NewWorker(MockSubmitter{}, config)
				if e != nil {
					t.Fatal(e)
				}

				if v.isValid {
					n, _ := strconv.Atoi(v.envValue)
					if worker.MetricsPerRequestSize != n {
						t.Fatal(fmt.Sprintf("Wrong metricsPerRequest size \n got: %d\n want: %d", worker.MetricsPerRequestSize, n))
					}
				} else {
					//should fallback to default value
					if worker.MetricsPerRequestSize != 1000 {
						t.Fatal(fmt.Sprintf("Wrong default metricsPerRequest size \n got: %d\n want: %d", worker.MetricsPerRequestSize, 1000))
					}
				}
			})
	}
}

func TestNewWorkerSubmitterNil(t *testing.T) {
	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(nil, config)

	if worker != nil {
		t.Fatalf("worker should be nill")
	}

	if err.Error() != "metrics submitter should not be nil" {
		t.Fatal(fmt.Sprintf("Wrong error message \n got: %q\n want: %q", err.Error(), "metrics submitter should not be nil"))
	}
}

func TestSubmitError(t *testing.T) {
	anodotSubmitterErrors.Reset()
	_ = os.Setenv("ANODOT_METRICS_PER_REQUEST_SIZE", "10")

	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}

	worker, err := NewWorker(MockSubmitter{f: func(anodot20Metrics []metrics.Anodot20Metric) (response metrics.AnodotResponse, e error) {
		return &metrics.CreateResponse{
			Errors:       nil,
			HttpResponse: &http.Response{StatusCode: 500},
		}, fmt.Errorf("error happened")
	}}, config)

	if err != nil {
		t.Fatal(err)
	}

	worker.Do(randomMetrics(10))
	waitWorkers(worker, 1)
	waitWorkers(worker, 0)
	v := testutil.ToFloat64(anodotSubmitterErrors)
	if v != 1 {
		t.Fatal(fmt.Sprintf("Wrong error counter \n got: %f\n want: %f", v, float64(1)))
	}

	const metadata = `
        # HELP anodot_server_http_responses_total Total number of HTTP responses of Anodot server
        # TYPE anodot_server_http_responses_total counter
	`

	expected := `

		anodot_server_http_responses_total{anodot_url="127.0.0.1",response_code="500"} 1
	`

	err = testutil.CollectAndCompare(serverHTTPResponses, strings.NewReader(metadata+expected), "anodot_server_http_responses_total")
	if err != nil {
		t.Fatalf(err.Error())
	}

}

func TestSubmitErrorInReponse(t *testing.T) {
	anodotSubmitterErrors.Reset()

	_ = os.Setenv("ANODOT_METRICS_PER_REQUEST_SIZE", "10")

	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}

	worker, err := NewWorker(MockSubmitter{f: func(anodot20Metrics []metrics.Anodot20Metric) (response metrics.AnodotResponse, e error) {

		anodotResponse := &metrics.CreateResponse{
			HttpResponse: nil,
		}
		anodotResponse.Errors = append(anodotResponse.Errors, struct {
			Description string
			Error       int64
			Index       string
		}{Description: "some text", Error: int64(2), Index: string('3')})

		return anodotResponse, fmt.Errorf(anodotResponse.ErrorMessage())

	}}, config)

	if err != nil {
		t.Fatal(err)
	}

	worker.Do(randomMetrics(10))
	waitWorkers(worker, 1)
	waitWorkers(worker, 0)
	v := testutil.ToFloat64(anodotSubmitterErrors)
	if v != 1 {
		t.Fatal(fmt.Sprintf("Wrong error counter \n got: %f\n want: %f", v, float64(1)))
	}
}

func TestNoMetricsSendInDebugMode(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	reqSize := 1500
	mockSubmitter := &MockSubmitter{f: func(metrics []metrics.Anodot20Metric) (metrics.AnodotResponse, error) {
		t.Errorf("No metrics should be sent in debug mode")
		return nil, nil
	}}

	config, err := NewWorkerConfig()
	if err != nil {
		t.Fatal(err)
	}

	worker, err := NewWorker(mockSubmitter, config)
	if err != nil {
		t.Fatal(err)
	}
	worker.Do(randomMetrics(reqSize))
}

type MockSubmitter struct {
	f func([]metrics.Anodot20Metric) (metrics.AnodotResponse, error)
}

func (m MockSubmitter) SubmitMetrics(metrics []metrics.Anodot20Metric) (metrics.AnodotResponse, error) {
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

func waitWorkers(w *Worker, excpected int64) {
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		if atomic.LoadInt64(&w.currentWorkers) == excpected {
			return
		}
	}
}
