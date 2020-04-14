package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/anodot/anodot-common/pkg/metrics"
	"github.com/anodot/anodot-remote-write/pkg/relabling"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	log "k8s.io/klog/v2"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	maxPropertyLength     = 150
	maxKeyLength          = 50
	maxNumberOfProperties = 20
	whatPropertyName      = "what"
	anodotTagLabelPrefix  = "anodot_tag_"
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

	k8sRelablingDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anodot_parser_kubernetes_relabling_metrics_dropped",
		Help: "Number of metrics dropped after ",
	})
)

var StatefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

type MetricsProcessor interface {
	Mutate(prometheusMetric model.Metric)
	Name() string
}

type KubernetesPodNameProcessor struct {
	PodsData *relabling.PodsMapping
}

func (k *KubernetesPodNameProcessor) Name() string {
	return "KubernetesPodNameProcessor"
}

//TODO what to do if pod and pod_name defined ?

func (k *KubernetesPodNameProcessor) Mutate(prometheusMetric model.Metric) {
	metricName := prometheusMetric[model.MetricNameLabel]

	_, podLabelExists := prometheusMetric["pod"]
	_, podNameLabelExists := prometheusMetric["pod_name"]

	if !podLabelExists && !podNameLabelExists {
		//nothing to do...
		log.V(4).Infof("no 'pod' or 'pod_name label exists for metrics %s'. skipping mutation", metricName)
		return
	}

	for labelName, labelValue := range prometheusMetric {
		if labelName == "pod" || labelName == "pod_name" {
			//do nothing for sts
			podName := string(labelValue)
			namespace := string(prometheusMetric["namespace"])
			if StatefulPodRegex.MatchString(podName) {
				continue
			}

			//check if pod is in excluded list
			anodotPodName := k.PodsData.ExcludedPods.Lookup(relabling.SearchEntry{
				PodName:   podName,
				Namespace: namespace,
			})
			//check in all namespaces also
			if anodotPodName == "" {
				anodotPodName = k.PodsData.ExcludedPods.LookupAllNamespaces(podName)
			}

			if anodotPodName != "" {
				log.V(4).Infof("pod %q is in excluded list..nothing to do", podName)
				//inc counter
				return
			}

			anodotPodName = k.PodsData.WhitelistedPods.Lookup(relabling.SearchEntry{
				PodName:   podName,
				Namespace: namespace,
			})
			if anodotPodName == "" {
				anodotPodName = k.PodsData.WhitelistedPods.LookupAllNamespaces(podName)
			}

			if anodotPodName == "" {
				//drop metrics..since we does not know anything about pods
				log.Warning(fmt.Sprintf("%q metrics is dropped. no %q labels present on pod %q in namespace=%s.", metricName, relabling.AnodotPodNameLabel, podName, namespace))
				removeMetricData(prometheusMetric)
				return
			}

			log.V(4).Infof("%s found pod name: %s", podName, anodotPodName)

			if len(strings.TrimSpace(anodotPodName)) != 0 {
				prometheusMetric[labelName] = model.LabelValue(anodotPodName)
				prometheusMetric[anodotTagLabelPrefix+"originalPodName"] = model.LabelValue(podName)
				log.V(4).Infof("set '%s' to='%s' for '%s'='%s'", relabling.AnodotPodNameLabel, anodotPodName, string(labelName), podName)
			} else {
				log.Warning("setting prometheus metric to nil ")
				removeMetricData(prometheusMetric)
				return
			}
			continue
		}
	}
}

type AnodotParser struct {
	FilterOutProperties map[string]string `json:"fop"`
	FilterInProperties  map[string]string `json:"fip"`

	// Anodot Metrics tags that will be assigned to all metrics.
	// https://support.anodot.com/hc/en-us/articles/360020259354-Posting-2-0-Metrics-
	Tags map[string]string

	MetricsProcessors []MetricsProcessor
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

func (p *AnodotParser) extractTags(prometheusMetric model.Metric) map[string]string {
	res := make(map[string]string)
	for k, v := range p.Tags {
		res[k] = v
	}

	for k, v := range prometheusMetric {
		if strings.HasPrefix(string(k), anodotTagLabelPrefix) {
			res[strings.TrimPrefix(string(k), anodotTagLabelPrefix)] = string(v)
		}
	}

	for k := range prometheusMetric {
		if strings.HasPrefix(string(k), anodotTagLabelPrefix) {
			delete(prometheusMetric, k)
		}
	}

	for k, v := range res {
		if len(v) >= maxPropertyLength {
			v = v[:maxPropertyLength]
		}

		if len(k) >= maxKeyLength {
			delete(res, k)
			k = k[:maxKeyLength]
		}
		res[k] = v
	}

	return res
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
			log.V(4).Infof("'%s' skipped. Nan and Inf values are ignored", r.Metric.String())
			incorrectValue.Inc()
			continue
		}

		if len(r.Metric) > maxNumberOfProperties {
			metricsPropertiesSizeExceeded.Inc()
			log.Warningf("Metric is skipped. Numer of lables=%d is more that allowed(%d). %s", len(r.Metric), maxNumberOfProperties, r)
			continue
		}

		for _, processor := range p.MetricsProcessors {
			processor.Mutate(r.Metric)
		}

		if len(r.Metric) == 0 {
			k8sRelablingDropped.Inc()
			log.Warningf("dropping empty metric %q", r.Metric[model.MetricNameLabel])
			continue
		}

		labels := make(model.LabelNames, 0, len(r.Metric))
		for l := range r.Metric {
			labels = append(labels, l)
		}
		sort.Sort(labels)
		metric.Properties = make(map[string]string)

		metric.Tags = p.extractTags(r.Metric)

		for _, l := range labels {

			v := r.Metric[l]

			if len(l) == 0 || len(v) == 0 {
				continue
			}

			if len(l) >= maxKeyLength {
				l = l[:maxKeyLength]
			}

			if len(v) >= maxPropertyLength {
				v = v[:maxPropertyLength]
			}

			if l == model.MetricNameLabel {
				metric.Properties[whatPropertyName] = string(v)
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

func removeMetricData(prometheusMetric model.Metric) {
	for name := range prometheusMetric {
		delete(prometheusMetric, name)
	}
}
