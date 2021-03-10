package prometheus

import (
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
)

func TestRelabel(t *testing.T) {
	tests := []struct {
		input   model.Metric
		relabel []*Config
		output  model.Metric
	}{
		{
			input: model.Metric{
				"a": "foo",
				"b": "bar",
				"c": "baz",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("f(.*)"),
					TargetLabel:  "d",
					Separator:    ";",
					Replacement:  "ch${1}-ch${1}",
					Action:       Replace,
				},
			},
			output: model.Metric{
				"a": "foo",
				"b": "bar",
				"c": "baz",
				"d": "choo-choo",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
				"b": "bar",
				"c": "baz",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a", "b"},
					Regex:        MustNewRegexp("f(.*);(.*)r"),
					TargetLabel:  "a",
					Separator:    ";",
					Replacement:  "b${1}${2}m", // boobam
					Action:       Replace,
				},
				{
					SourceLabels: model.LabelNames{"c", "a"},
					Regex:        MustNewRegexp("(b).*b(.*)ba(.*)"),
					TargetLabel:  "d",
					Separator:    ";",
					Replacement:  "$1$2$2$3",
					Action:       Replace,
				},
			},
			output: model.Metric{
				"a": "boobam",
				"b": "bar",
				"c": "baz",
				"d": "boooom",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp(".*o.*"),
					Action:       Drop,
				}, {
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("f(.*)"),
					TargetLabel:  "d",
					Separator:    ";",
					Replacement:  "ch$1-ch$1",
					Action:       Replace,
				},
			},
			output: nil,
		},
		{
			input: model.Metric{
				"a": "foo",
				"b": "bar",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp(".*o.*"),
					Action:       Drop,
				},
			},
			output: nil,
		},
		{
			input: model.Metric{
				"a": "abc",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp(".*(b).*"),
					TargetLabel:  "d",
					Separator:    ";",
					Replacement:  "$1",
					Action:       Replace,
				},
			},
			output: model.Metric{
				"a": "abc",
				"d": "b",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("no-match"),
					Action:       Drop,
				},
			},
			output: model.Metric{
				"a": "foo",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("f|o"),
					Action:       Drop,
				},
			},
			output: model.Metric{
				"a": "foo",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("no-match"),
					Action:       Keep,
				},
			},
			output: nil,
		},
		{
			input: model.Metric{
				"a": "foo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("f.*"),
					Action:       Keep,
				},
			},
			output: model.Metric{
				"a": "foo",
			},
		},
		{
			// No replacement must be applied if there is no match.
			input: model.Metric{
				"a": "boo",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("f"),
					TargetLabel:  "b",
					Replacement:  "bar",
					Action:       Replace,
				},
			},
			output: model.Metric{
				"a": "boo",
			},
		},
		{
			input: model.Metric{
				"a": "foo",
				"b": "bar",
				"c": "baz",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"c"},
					TargetLabel:  "d",
					Separator:    ";",
					Action:       HashMod,
					Modulus:      1000,
				},
			},
			output: model.Metric{
				"a": "foo",
				"b": "bar",
				"c": "baz",
				"d": "976",
			},
		},
		{
			input: model.Metric{
				"a":  "foo",
				"b1": "bar",
				"b2": "baz",
			},
			relabel: []*Config{
				{
					Regex:       MustNewRegexp("(b.*)"),
					Replacement: "bar_${1}",
					Action:      LabelMap,
				},
			},
			output: model.Metric{
				"a":      "foo",
				"b1":     "bar",
				"b2":     "baz",
				"bar_b1": "bar",
				"bar_b2": "baz",
			},
		},
		{
			input: model.Metric{
				"a":             "foo",
				"__meta_my_bar": "aaa",
				"__meta_my_baz": "bbb",
				"__meta_other":  "ccc",
			},
			relabel: []*Config{
				{
					Regex:       MustNewRegexp("__meta_(my.*)"),
					Replacement: "${1}",
					Action:      LabelMap,
				},
			},
			output: model.Metric{
				"a":             "foo",
				"__meta_my_bar": "aaa",
				"__meta_my_baz": "bbb",
				"__meta_other":  "ccc",
				"my_bar":        "aaa",
				"my_baz":        "bbb",
			},
		},
		{ // valid case
			input: model.Metric{
				"a": "some-name-value",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("some-([^-]+)-([^,]+)"),
					Action:       Replace,
					Replacement:  "${2}",
					TargetLabel:  "${1}",
				},
			},
			output: model.Metric{
				"a":    "some-name-value",
				"name": "value",
			},
		},
		{ // invalid replacement ""
			input: model.Metric{
				"a": "some-name-value",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("some-([^-]+)-([^,]+)"),
					Action:       Replace,
					Replacement:  "${3}",
					TargetLabel:  "${1}",
				},
			},
			output: model.Metric{
				"a": "some-name-value",
			},
		},
		{ // invalid target_labels
			input: model.Metric{
				"a": "some-name-value",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("some-([^-]+)-([^,]+)"),
					Action:       Replace,
					Replacement:  "${1}",
					TargetLabel:  "${3}",
				},
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("some-([^-]+)-([^,]+)"),
					Action:       Replace,
					Replacement:  "${1}",
					TargetLabel:  "0${3}",
				},
				{
					SourceLabels: model.LabelNames{"a"},
					Regex:        MustNewRegexp("some-([^-]+)-([^,]+)"),
					Action:       Replace,
					Replacement:  "${1}",
					TargetLabel:  "-${3}",
				},
			},
			output: model.Metric{
				"a": "some-name-value",
			},
		},
		{ // more complex real-life like usecase
			input: model.Metric{
				"__meta_sd_tags": "path:/secret,job:some-job,label:foo=bar",
			},
			relabel: []*Config{
				{
					SourceLabels: model.LabelNames{"__meta_sd_tags"},
					Regex:        MustNewRegexp("(?:.+,|^)path:(/[^,]+).*"),
					Action:       Replace,
					Replacement:  "${1}",
					TargetLabel:  "__metrics_path__",
				},
				{
					SourceLabels: model.LabelNames{"__meta_sd_tags"},
					Regex:        MustNewRegexp("(?:.+,|^)job:([^,]+).*"),
					Action:       Replace,
					Replacement:  "${1}",
					TargetLabel:  "job",
				},
				{
					SourceLabels: model.LabelNames{"__meta_sd_tags"},
					Regex:        MustNewRegexp("(?:.+,|^)label:([^=]+)=([^,]+).*"),
					Action:       Replace,
					Replacement:  "${2}",
					TargetLabel:  "${1}",
				},
			},
			output: model.Metric{
				"__meta_sd_tags":   "path:/secret,job:some-job,label:foo=bar",
				"__metrics_path__": "/secret",
				"job":              "some-job",
				"foo":              "bar",
			},
		},
		{
			input: model.Metric{
				"a":  "foo",
				"b1": "bar",
				"b2": "baz",
			},
			relabel: []*Config{
				{
					Regex:  MustNewRegexp("(b.*)"),
					Action: LabelKeep,
				},
			},
			output: model.Metric{
				"b1": "bar",
				"b2": "baz",
			},
		},
		{
			input: model.Metric{
				"a":  "foo",
				"b1": "bar",
				"b2": "baz",
			},
			relabel: []*Config{
				{
					Regex:  MustNewRegexp("(b.*)"),
					Action: LabelDrop,
				},
			},
			output: model.Metric{
				"a": "foo",
			},
		},
	}

	for _, test := range tests {
		res := Process(test.input, test.relabel...)
		if !reflect.DeepEqual(test.output, res) {
			t.Fatalf("\033[31m\nexp: %#v\n\ngot: %#v\033[39m\n", test.output, res)
		}
	}
}

func TestLoadFile(t *testing.T) {
	expectedRelabel := MetricRelabel{
		Configs: []*Config{
			{
				SourceLabels: model.LabelNames{"job", "__meta_dns_name"},
				TargetLabel:  "job",
				Separator:    ";",
				Regex:        MustNewRegexp("(.*)some-[regex]"),
				Replacement:  "foo-${1}",
				Action:       Replace,
			},
			{
				SourceLabels: model.LabelNames{"__name__"},
				Separator:    ";",
				Regex:        MustNewRegexp("expensive.*"),
				Replacement:  "$1",
				Action:       Drop,
			},

			{
				SourceLabels: model.LabelNames{"__name__"},
				Separator:    ";",
				Regex:        MustNewRegexp("(.*)"),
				Replacement:  "true",
				TargetLabel:  "anodot_include",
				Action:       Replace,
			},
		},
	}

	relabelConfig, err := NewMetricRelabel("./test_data/relabel_config.yaml")
	if err != nil {
		t.Fatal(err.Error())
	}

	expected := expectedRelabel.Configs
	actual := relabelConfig.Configs
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\033[31m\nexp: %#v\n\ngot: %#v\033[39m\n", expected, actual)
	}

}
