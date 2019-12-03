package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"testing"
	"time"
)

const expectedNumberOfMetrics int = 4

type Metrics struct {
	Properties map[string]string `json:"properties"`
	Timestamp  float64           `json:"timestamp"`
	Value      float64           `json:"value"`
	Tags       struct {
	} `json:"tags"`
}

func TestMain(m *testing.M) {
	out, err := shellCommand("docker-compose", "down", "-v")
	log.Println(string(out))
	if err != nil {
		log.Fatal("Failed to execute docker-compose command")
	}

	err = os.RemoveAll("./tmp")
	if err != nil {
		log.Fatal("Failed to cleanup './tmp' folder:", err)
	}

	err = os.Mkdir("./tmp", 0777)
	if err != nil {
		log.Fatal("Failed to create './tmp' folder", err)
	}

	out, err = shellCommand("docker-compose", "up", "-d")
	log.Println(string(out))
	if err != nil {
		log.Fatal("Failed to execute docker-compose command:", err)
	}
	//need to wait until metrics will be sent to remote write.
INFINITE_LOOP:
	for {
		samples, err := metrics("http://localhost:1234/metrics", 3, 1*time.Second)
		if err != nil {
			log.Fatal("Failed to get metrics:", err)
		}

		for _, s := range samples {
			if s.Metric[model.MetricNameLabel] == "anodot_remote_write_prometheus_samples_received_total" {
				if s.Value.Equal(model.SampleValue(expectedNumberOfMetrics)) {
					break INFINITE_LOOP
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	out, err = shellCommand("docker-compose", "logs")
	log.Println(string(out))
	if err != nil {
		log.Fatal("Failed to execute docker-compose command", err)
	}

	out, err = shellCommand("docker", "cp", "anodot-metrics-stub:/tmp/metrics.log", "./tmp/metrics.log")
	log.Println(string(out))
	if err != nil {
		log.Fatal("Failed to get metrics logs", err)
	}

	out, err = shellCommand("docker-compose", "down", "-v")
	log.Println(string(out))
	if err != nil {
		log.Fatal("Failed to execute docker-compose command:", err)
	}

	run := m.Run()
	err = os.RemoveAll("./tmp")
	if err != nil {
		log.Fatal("Failed to cleanup './tmp' folder:", err)
	}

	os.Exit(run)
}

func metrics(url string, retries int, timeout time.Duration) ([]*model.Sample, error) {
	var (
		samples  []*model.Sample
		err      error
		response *http.Response
	)
	for retries > 0 {
		response, err = http.Get(url)
		if err != nil {
			retries -= 1
			time.Sleep(timeout)
		} else {
			break
		}
	}

	if err != nil {
		return samples, err
	}

	dec := expfmt.NewDecoder(response.Body, expfmt.FmtText)
	decoder := expfmt.SampleDecoder{
		Dec:  dec,
		Opts: &expfmt.DecodeOptions{},
	}

	for {
		var v model.Vector
		if err := decoder.Decode(&v); err != nil {
			if err == io.EOF {
				// Expected loop termination condition.
				return samples, nil
			}
			return nil, err
		}
		samples = append(samples, v...)
	}
}

func shellCommand(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s failed with %s\n", command, err)
	}
	return out, err
}

func TestMetricsSize(t *testing.T) {
	data, err := ioutil.ReadFile("./tmp/metrics.log")
	if err != nil {
		t.Fatal("Failed to read metrics output file:", err)
	}

	metrics := make([]Metrics, 0)
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal("Failed to read metrics output as json:", err)
	}

	if len(metrics) != 4 {
		t.Fatal(fmt.Sprintf("Not expected numer of metrics.\n Got: %v\n Want: %v", len(metrics), 4))
	}

}

func TestMetricsSortedASC(t *testing.T) {
	data, err := ioutil.ReadFile("./tmp/metrics.log")
	if err != nil {
		t.Fatal("Failed to read metrics output file:", err)
	}

	metrics := make([]Metrics, 0)
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal("Failed to read metrics output as json:", err)
	}

	sortedASC := sort.SliceIsSorted(metrics, func(i, j int) bool {
		return metrics[i].Timestamp < metrics[j].Timestamp
	})

	if !sortedASC {
		t.Fatal("Metrics should be sorted in ASC way.")
	}
}

func TestMetricsData(t *testing.T) {
	data, err := ioutil.ReadFile("./tmp/metrics.log")
	if err != nil {
		t.Fatal("Failed to read metrics output file:", err)
	}

	metrics := make([]Metrics, 0)
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal("Failed to read metrics output as json:", err)
	}

	if len(metrics) != expectedNumberOfMetrics {
		t.Fatal("Number of metrics does not match")
	}

	for _, m := range metrics {
		what := m.Properties["what"]
		switch what {
		case "exported_http_requests_total":

			expectedProperties := map[string]string{
				"code":     "200",
				"instance": "anodot-metrics-stub:8080",
				"job":      "test-app",
				"method":   "get",
				"what":     "exported_http_requests_total",
			}

			expectedValue := float64(1)
			actualValue := m.Value
			if actualValue != expectedValue {
				t.Fatal(fmt.Sprintf("Wrong value for metric %s,\n Got %v\n want %v", what, actualValue, expectedValue))
			}

			if !reflect.DeepEqual(m.Properties, expectedProperties) {
				t.Fatal(fmt.Sprintf("Not equal properties.\n Got %v\n, want %v", m.Properties, expectedProperties))
			}
		case "exported_version":
			expectedProperties := map[string]string{
				"instance": "anodot-metrics-stub:8080",
				"job":      "test-app",
				"version":  "v0%2E1%2E0",
				"what":     "exported_version",
			}

			expectedValue := 0.1
			actualValue := m.Value
			if actualValue != expectedValue {
				t.Fatal(fmt.Sprintf("Wrong value for metric %s,\n Got %v\n want %v", what, actualValue, expectedValue))
			}

			if !reflect.DeepEqual(m.Properties, expectedProperties) {
				t.Fatal(fmt.Sprintf("Not equal properties.\n Got %v\n want %v", m.Properties, expectedProperties))
			}

		default:
			t.Fatal(fmt.Sprintf("Unsupported what value=%q", what))
		}
	}
}
