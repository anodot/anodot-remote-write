package anodotParser

import (
	"github.com/prometheus/common/model"
	"fmt"
	"sort"
	"bytes"
	"strings"
	"math"
	"github.com/anodot/anodot-common/remoteStats"
	"encoding/json"
	"log"
	"errors"
)

type AnodotMetric struct {
	Properties  map[string]string   `json:"properties"`
	Timestamp   float64   			`json:"timestamp"`
	Value 		float64 			`json:"value"`
	Tags        map[string]string	`json:"tags"`
}
type AnodotParser struct {
	FilterOutProperties map[string]string `json:"fop"`
	FilterInProperties map[string]string `json:"fip"`
}

const (
	MAX_PROPERTY_LENGTH = 150
	MAX_NUMBER_OF_PROPERTIES = 20
	WHAT_PROPERTY = "what"
	TARGET_TYPE = "target_type"
	COUNTER = "COUNTER"
	GAUGE = "GAUGE"
)  


const (
	symbols    = "(){},=.'\"\\"
	printables = ("0123456789abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"!\"#$%&\\'()*+,-./:;<=>?@[\\]^_`{|}~")
)

func NewAnodotParser(filterIn *string, filterOut *string) (error,AnodotParser) {

	var parser AnodotParser

	if filterIn != nil && *filterIn != "" {
		error := json.Unmarshal([]byte(*filterIn),&parser.FilterInProperties)
		if error != nil {
			log.Println("Failed to Parse In Filter")
			return errors.New("Failed to Parse Filter"),parser
		}
	}

	if filterOut != nil && *filterOut != "" {
		error := json.Unmarshal([]byte(*filterOut),&parser.FilterOutProperties)
		if error != nil {
			log.Println("Failed to Parse Out Filter")
			return errors.New("Failed to Parse Filter"),parser
		}
	}
	return nil,parser
}

func (p *AnodotParser) filter(metrics *[]AnodotMetric,metric *AnodotMetric) {

	for k := range p.FilterInProperties{
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

	for k := range p.FilterOutProperties{
		if val, ok := metric.Properties[k]; ok {
			if p.FilterOutProperties[k] == val {
				return
			}
		}
	}

	*metrics = append(*metrics, *metric)
}


func (p *AnodotParser) escape(tv model.LabelValue) string {
	length := len(tv)
	result := bytes.NewBuffer(make([]byte, 0, length))
	for i := 0; i < length; i++ {
		b := tv[i]
		switch {
		// . is reserved by graphite, % is used to escape other bytes.
		case b == '.' || b == '%' || b == '/' || b == '=':
			fmt.Fprintf(result, "%%%X", b)
			// These symbols are ok only if backslash escaped.
		case strings.IndexByte(symbols, b) != -1:
			result.WriteString("\\" + string(b))
			// These are all fine.
		case strings.IndexByte(printables, b) != -1:
			result.WriteByte(b)
			// Defaults to percent-encoding.
		default:
			fmt.Fprintf(result, "%%%X", b)
		}
	}

	if len(result.String()) >= MAX_PROPERTY_LENGTH{
		return result.String()[:MAX_PROPERTY_LENGTH]
	}

	return result.String()
}


func (p *AnodotParser) ParsePrometheusRequest(samples model.Samples, stats remoteStats.RemoteStatsInterface)([]AnodotMetric)  {

	metrics := make([]AnodotMetric,0)

	for _, r := range samples {
		var metric AnodotMetric
		metric.Timestamp = float64(r.Timestamp.UnixNano()) / 1e9
		metric.Value = float64(r.Value)

		if math.IsNaN(metric.Value) || math.IsInf(metric.Value, 0) {
			continue
		}

		labels := make(model.LabelNames, 0, len(r.Metric))
		for l := range r.Metric {
			labels = append(labels, l)
		}
		sort.Sort(labels)
		metric.Properties = make(map[string]string)
		for _, l := range labels {

			if len(labels) > MAX_NUMBER_OF_PROPERTIES{
				stats.UpdateMeter(remoteStats.PARSING_ERRORS,1)
				continue
			}

			v := r.Metric[l]

			if  len(l) == 0 || len(v) == 0{
				continue
			}

			if len(l) >= MAX_PROPERTY_LENGTH{
				l = l[:MAX_PROPERTY_LENGTH]
			}


			if l == model.MetricNameLabel{
				metric.Properties[WHAT_PROPERTY] = p.escape(v)
				metric.Tags = make(map[string]string)
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

			metric.Properties[string(l)] = p.escape(v);
		}
		p.filter(&metrics, &metric)

	}
	return metrics
}






