package anodotRemoteTests

import (
	"testing"
	"github.com/prometheus/common/model"
	"math"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/remoteStats"
)

func TestReceiver(t *testing.T) {

	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value1",
			},
			Timestamp: model.Time(123456789123),
			Value:     333,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789123),
			Value:     8.14,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "test3",
			},
			Timestamp: model.Time(123456789123),
			Value:     6.15,
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

	err,parser := anodotParser.NewAnodotParser(nil,nil)

	if err != nil{
		t.Fail()
	}

	var stats remoteStats.MockRemoteStats
	metrics := parser.ParsePrometheusRequest(samples,&stats)

	if metrics == nil || len(metrics) == 0{
		t.Fail()
	}

	for _,m := range metrics{
		_,ok := m.Properties["what"]
		if !ok{
			t.Fail()
		}
	}
}

func TestFilters(t *testing.T) {
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

	filterOut :=  `{"test_label":"test_label_value2"}`
	err,parser := anodotParser.NewAnodotParser(nil,&filterOut)

	if err != nil{
		t.Fail()
	}


	var stats remoteStats.MockRemoteStats
	metrics := parser.ParsePrometheusRequest(samples,&stats)

	if len(metrics) > 3{
		t.Fail()
	}

}

func TestFilters2(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value1",
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

	filterIn :=  `{"test_label":"test_label_value2"}`
	err,parser := anodotParser.NewAnodotParser(&filterIn,nil)

	if err != nil{
		t.Fail()
	}


	var stats remoteStats.MockRemoteStats
	metrics := parser.ParsePrometheusRequest(samples,&stats)

	if len(metrics) > 1{
		t.Fail()
	}

}

func TestTargetType(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric_total",
				"test_label":          "test_label_value1",
			},
			Timestamp: model.Time(123456789123),
			Value:     1.11,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric_count",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789123),
			Value:     2,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric_sum",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789123),
			Value:     2,
		},
	}

	err,parser := anodotParser.NewAnodotParser(nil,nil)

	if err != nil{
		t.Fail()
	}

	var stats remoteStats.MockRemoteStats
	metrics := parser.ParsePrometheusRequest(samples,&stats)
	if metrics[0].Tags[anodotParser.TARGET_TYPE] != anodotParser.COUNTER{
		t.Fail()
	}
	if metrics[1].Tags[anodotParser.TARGET_TYPE] != anodotParser.COUNTER{
		t.Fail()
	}
	if metrics[2].Tags[anodotParser.TARGET_TYPE] != anodotParser.COUNTER{
		t.Fail()
	}
}