package prometheus

import (
	"bytes"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/model"
	"math"
	"testing"
)

func TestTags(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "with_tags",
			},
			Timestamp: model.Time(1574693483),
			Value:     1,
		},
	}

	parser, err := NewAnodotParser(nil, nil, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	if len(metrics) != 1 {
		t.Fatalf("unexpected number of metrics")
	}

	for _, m := range metrics {
		if !(m.Tags["key"] == "value") {
			t.Fatalf("tags should be set correctly")
		}
	}
}

func TestConstructorFilterIn(t *testing.T) {
	s := "invalid_json"
	_, err := NewAnodotParser(&s, nil, nil)
	if err == nil {
		t.Fatalf("error should be retuned")
	}

	if err.Error() != "failed to parse filterIn expression" {
		t.Fatal(fmt.Sprintf("Wrong error message \n got: %s\n want: %s", err.Error(), "failed to parse filterIn expression"))
	}
}

func TestConstructorFilterOut(t *testing.T) {
	s := "invalid_json"
	_, err := NewAnodotParser(nil, &s, nil)
	if err == nil {
		t.Fatalf("error should be retuned")
	}

	if err.Error() != "failed to parse filterOut expression" {
		t.Fatal(fmt.Sprintf("Wrong error message \n got: %s\n want: %s", err.Error(), "failed to parse filterOut expression"))
	}
}

func TestNotAcceptableValue(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "pos_inf_value",
			},
			Timestamp: model.Time(1574693483),
			Value:     model.SampleValue(math.Inf(1)),
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "neg_inf_value",
			},
			Timestamp: model.Time(1574693483),
			Value:     model.SampleValue(math.Inf(-1)),
		},
	}

	parser, err := NewAnodotParser(nil, nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	if len(metrics) != 0 {
		t.Fatalf("metrics with +Inf and -Inf values should not be accepted")
	}

	v := testutil.ToFloat64(incorrectValue)
	if v != 2 {
		t.Fatal(fmt.Sprintf("Wrong error counter \n got: %f\n want: %f", v, float64(2)))
	}
}

func TestMetricsMaxPropertiesSize(t *testing.T) {

	m := make(map[model.LabelName]model.LabelValue, 20)
	m[model.MetricNameLabel] = "more_than_20_props"

	for i := 0; i < 20; i++ {
		m[model.LabelName(fmt.Sprintf("key_%d", i))] = "test"
	}

	samples := model.Samples{
		{
			Metric:    m,
			Timestamp: model.Time(1574693483),
			Value:     3,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "ok_value",
			},
			Timestamp: model.Time(1574693483),
			Value:     4,
		},
	}

	parser, err := NewAnodotParser(nil, nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	if len(metrics) != 1 {
		t.Fatalf("metric with more than 20 label values are not accepted by anodot")
	}

	v := testutil.ToFloat64(metricsPropertiesSizeExceeded)
	if v != 1 {
		t.Fatal(fmt.Sprintf("Wrong error counter \n got: %f\n want: %f", v, float64(1)))
	}
}

func TestMaxPropertyLength(t *testing.T) {
	var longKey bytes.Buffer
	for i := 1; i <= 60; i++ {
		longKey.WriteString("a")
	}

	var longValue bytes.Buffer
	for i := 1; i <= 160; i++ {
		longValue.WriteString("a")
	}

	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "long_value",
				"long_value":          model.LabelValue(longValue.String()),
			},
			Timestamp: model.Time(1574693483),
			Value:     1,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel:             "long_key",
				model.LabelName(longKey.String()): "long_key",
			},
			Timestamp: model.Time(1574693483),
			Value:     2,
		},
	}

	parser, err := NewAnodotParser(nil, nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	if len(metrics) != 2 {
		t.Fatalf("invalid number of metrics")
	}

	for _, m := range metrics {
		for k, v := range m.Properties {
			if len(k) > 50 {
				t.Fatalf("metrics property key should be <=50")
			}

			if len(v) > 150 {
				t.Fatalf("metrics property value should be <=150")
			}
		}
	}
}

func TestReceiver(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value1",
			},
			Timestamp: model.Time(1574693483),
			Value:     333,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(1574693483),
			Value:     8.14,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "test3",
			},
			Timestamp: model.Time(1574693483),
			Value:     6.15,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "pos_inf_value",
			},
			Timestamp: model.Time(1574693483),
			Value:     model.SampleValue(math.Inf(1)),
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "neg_inf_value",
			},
			Timestamp: model.Time(1574693483),
			Value:     model.SampleValue(math.Inf(-1)),
		},
	}

	parser, err := NewAnodotParser(nil, nil, nil)
	if err != nil {
		t.Error(err)
	}

	anodotMetrics := parser.ParsePrometheusRequest(samples)

	if len(anodotMetrics) != 3 {
		t.Fatalf(fmt.Sprintf("Expected number of metrics=3. Found=%d", len(anodotMetrics)))
	}

	for _, m := range anodotMetrics {
		_, ok := m.Properties["what"]
		if !ok {
			t.Fatalf(fmt.Sprintf("no what propertry for metric %+v\n", m))
		}
	}
}

func TestFilterOut(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value1",
			},
			Timestamp: model.Time(123456789123),
			Value:     13,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789123),
			Value:     0.99993,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "test3",
			},
			Timestamp: model.Time(123456789123),
			Value:     86.1234,
		},
	}

	filterOut := `{"test_label":"test_label_value2"}`
	parser, err := NewAnodotParser(nil, &filterOut, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	for _, v := range metrics {
		if v.Properties["test_label"] == "test_label_value2" {
			t.Fatalf("metrics should be filtered out: %v ", v)
		}
	}
}

func TestFilterIn(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"tst_label":           "test_label_value1",
			},
			Timestamp: model.Time(123456789123),
			Value:     1.11,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789123),
			Value:     2,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "test3",
			},
			Timestamp: model.Time(123456789123),
			Value:     0,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "pos_inf_value",
			},
			Timestamp: model.Time(123456789123),
			Value:     model.SampleValue(math.Inf(1)),
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "neg_inf_value",
			},
			Timestamp: model.Time(123456789123),
			Value:     model.SampleValue(math.Inf(-1)),
		},
	}

	filterIn := `{"test_label":"test_label_value2","tst_label":"test_label_value1"}`
	parser, err := NewAnodotParser(&filterIn, nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	metrics := parser.ParsePrometheusRequest(samples)
	if len(metrics) != 2 {
		t.Fatalf("unexpected number of metrics")
	}
}

func TestMutation(t *testing.T) {
	var prometheusMetric model.Metric

	prometheusMetric = map[model.LabelName]model.LabelValue{"key": "value"}

	processor := KubernetesPodNameProcessor{}

	processor.Mutate(prometheusMetric)

	fmt.Println(prometheusMetric)
}
