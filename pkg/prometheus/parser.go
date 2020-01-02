package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	"log"
	"math"
	"sort"
)

const (
	maxPropertyLength     = 150
	maxNumberOfProperties = 20
)

var (
	metricsPropertiesSizeExceeded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_parser_max_number_labels_reached",
		Help: fmt.Sprintf("Number of times when Prometheus metric had more labels that allowed(%d).", maxNumberOfProperties),
	})

	incorrectValue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_parser_value_not_accepted",
		Help: "Number of times metrics value was not accepted",
	})
)

type AnodotParser struct {
	FilterOutProperties map[string]string `json:"fop"`
	FilterInProperties  map[string]string `json:"fip"`

	// Anodot Metrics tags that will be assigned to all metrics.
	// https://support.anodot.com/hc/en-us/articles/360020259354-Posting-2-0-Metrics-
	Tags map[string]string
}

func NewAnodotParser(filterIn *string, filterOut *string, tags map[string]string) (*AnodotParser, error) {
	var parser AnodotParser

	if filterIn != nil && *filterIn != "" {
		err := json.Unmarshal([]byte(*filterIn), &parser.FilterInProperties)
		if err != nil {
			return nil, errors.New("failed to parse filterIn expression")
		}
	}

	if filterOut != nil && *filterOut != "" {
		err := json.Unmarshal([]byte(*filterOut), &parser.FilterOutProperties)
		if err != nil {
			return nil, errors.New("failed to parse filterOut expression")
		}
	}

	parser.Tags = tags
	if parser.Tags == nil {
		parser.Tags = map[string]string{}
	}
	return &parser, nil
}

func (p *AnodotParser) filter(metrics *[]metrics.Anodot20Metric, metric *metrics.Anodot20Metric) {

	for k := range p.FilterInProperties {
		if val, ok := metric.Properties[k]; ok {
			if p.FilterInProperties[k] == val {
				*metrics = append(*metrics, *metric)
				return
			}
		}
	}

	if p.FilterInProperties != nil {
		return
	}

	for k := range p.FilterOutProperties {
		if val, ok := metric.Properties[k]; ok {
			if p.FilterOutProperties[k] == val {
				return
			}
		}
	}

	*metrics = append(*metrics, *metric)
}

func (p *AnodotParser) ParsePrometheusRequest(samples model.Samples) []metrics.Anodot20Metric {
	result := make([]metrics.Anodot20Metric, 0)

	for _, r := range samples {
		var metric metrics.Anodot20Metric

		metric.Timestamp = metrics.AnodotTimestamp{Time: r.Timestamp.Time()}
		metric.Value = float64(r.Value)

		if math.IsNaN(metric.Value) || math.IsInf(metric.Value, 0) {
			incorrectValue.Inc()
			continue
		}

		if len(r.Metric) > maxNumberOfProperties {
			metricsPropertiesSizeExceeded.Inc()
			log.Println(fmt.Sprintf("[WARNING]: Metric is skipped. Numer of lables=%d is more that allowed(%d). %s", len(r.Metric), maxNumberOfProperties, r))
			continue
		}

		labels := make(model.LabelNames, 0, len(r.Metric))
		for l := range r.Metric {
			labels = append(labels, l)
		}
		sort.Sort(labels)
		metric.Properties = make(map[string]string)
		metric.Tags = p.Tags

		for _, l := range labels {

			v := r.Metric[l]

			if len(l) == 0 || len(v) == 0 {
				continue
			}

			if len(l) >= maxPropertyLength {
				l = l[:maxPropertyLength]
			}

			if l == model.MetricNameLabel {
				metric.Properties["what"] = string(v)
				//Should be managed on prometheus config
				/*if strings.HasSuffix(metric.Properties[WHAT_PROPERTY],"_total") {
					metric.Properties[TARGET_TYPE] = COUNTER
				}else
				if strings.HasSuffix(metric.Properties[WHAT_PROPERTY],"_sum") {
					metric.Properties[TARGET_TYPE] = COUNTER
				}else
				if strings.HasSuffix(metric.Properties[WHAT_PROPERTY],"_count") {
					metric.Properties[TARGET_TYPE] = COUNTER
				}*/
				continue
			}
			metric.Properties[string(l)] = string(v)
		}
		p.filter(&result, &metric)

	}
	return result
}
