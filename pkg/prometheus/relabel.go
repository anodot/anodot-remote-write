package prometheus

import (
	"crypto/md5"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
)

var (
	relabelTarget = regexp.MustCompile(`^(?:(?:[a-zA-Z_]|\$(?:\{\w+\}|\w+))+\w*)+$`)

	DefaultRelabelConfig = Config{
		Action:      Replace,
		Separator:   ";",
		Regex:       MustNewRegexp("(.*)"),
		Replacement: "$1",
	}
)

// Action is the action to be performed on relabeling.
type Action string

const (
	// Replace performs a regex replacement.
	Replace Action = "replace"
	// Keep drops targets for which the input does not match the regex.
	Keep Action = "keep"
	// Drop drops targets for which the input does match the regex.
	Drop Action = "drop"
	// HashMod sets a label to the modulus of a hash of labels.
	HashMod Action = "hashmod"
	// LabelMap copies labels to other labelnames based on a regex.
	LabelMap Action = "labelmap"
	// LabelDrop drops any label matching the regex.
	LabelDrop Action = "labeldrop"
	// LabelKeep drops any label not matching the regex.
	LabelKeep Action = "labelkeep"
)

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (a *Action) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	switch act := Action(strings.ToLower(s)); act {
	case Replace, Keep, Drop, HashMod, LabelMap, LabelDrop, LabelKeep:
		*a = act
		return nil
	}
	return errors.Errorf("unknown relabel action %q", s)
}

type MetricRelabel struct {
	Configs []*Config `yaml:"relabel_configs"`
}

func NewMetricRelabel(configPath string) (*MetricRelabel, error) {
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var conf MetricRelabel
	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing YAML file %s", configPath)
	}

	return &conf, nil
}

// Config is the configuration for relabeling of target label sets.
type Config struct {
	// A list of labels from which values are taken and concatenated
	// with the configured separator in order.
	SourceLabels model.LabelNames `yaml:"source_labels,flow,omitempty"`
	// Separator is the string between concatenated values from the source labels.
	Separator string `yaml:"separator,omitempty"`
	// Regex against which the concatenation is matched.
	Regex Regexp `yaml:"regex,omitempty"`
	// Modulus to take of the hash of concatenated values from the source labels.
	Modulus uint64 `yaml:"modulus,omitempty"`
	// TargetLabel is the label to which the resulting string is written in a replacement.
	// Regexp interpolation is allowed for the replace action.
	TargetLabel string `yaml:"target_label,omitempty"`
	// Replacement is the regex replacement pattern to be used.
	Replacement string `yaml:"replacement,omitempty"`
	// Action is the action to be performed for the relabeling.
	Action Action `yaml:"action,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultRelabelConfig
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	if c.Regex.Regexp == nil {
		c.Regex = MustNewRegexp("")
	}
	if c.Modulus == 0 && c.Action == HashMod {
		return errors.Errorf("relabel configuration for hashmod requires non-zero modulus")
	}
	if (c.Action == Replace || c.Action == HashMod) && c.TargetLabel == "" {
		return errors.Errorf("relabel configuration for %s action requires 'target_label' value", c.Action)
	}
	if c.Action == Replace && !relabelTarget.MatchString(c.TargetLabel) {
		return errors.Errorf("%q is invalid 'target_label' for %s action", c.TargetLabel, c.Action)
	}
	if c.Action == LabelMap && !relabelTarget.MatchString(c.Replacement) {
		return errors.Errorf("%q is invalid 'replacement' for %s action", c.Replacement, c.Action)
	}
	if c.Action == HashMod && !model.LabelName(c.TargetLabel).IsValid() {
		return errors.Errorf("%q is invalid 'target_label' for %s action", c.TargetLabel, c.Action)
	}

	if c.Action == LabelDrop || c.Action == LabelKeep {
		if c.SourceLabels != nil ||
			c.TargetLabel != DefaultRelabelConfig.TargetLabel ||
			c.Modulus != DefaultRelabelConfig.Modulus ||
			c.Separator != DefaultRelabelConfig.Separator ||
			c.Replacement != DefaultRelabelConfig.Replacement {
			return errors.Errorf("%s action requires only 'regex', and no other fields", c.Action)
		}
	}

	return nil
}

// Regexp encapsulates a regexp.Regexp and makes it YAML marshalable.
type Regexp struct {
	*regexp.Regexp
	original string
}

// NewRegexp creates a new anchored Regexp and returns an error if the
// passed-in regular expression does not compile.
func NewRegexp(s string) (Regexp, error) {
	regex, err := regexp.Compile("^(?:" + s + ")$")
	return Regexp{
		Regexp:   regex,
		original: s,
	}, err
}

// MustNewRegexp works like NewRegexp, but panics if the regular expression does not compile.
func MustNewRegexp(s string) Regexp {
	re, err := NewRegexp(s)
	if err != nil {
		panic(err)
	}
	return re
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (re *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	r, err := NewRegexp(s)
	if err != nil {
		return err
	}
	*re = r
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (re Regexp) MarshalYAML() (interface{}, error) {
	if re.original != "" {
		return re.original, nil
	}
	return nil, nil
}

// Process returns a relabeled copy of the given label set. The relabel configurations
// are applied in order of input.
// If a label set is dropped, nil is returned.
// May return the input labelSet modified.
func Process(metric model.Metric, cfgs ...*Config) model.Metric {
	for _, cfg := range cfgs {
		metric = relabel(metric, cfg)
		if metric == nil {
			return nil
		}
	}
	return metric
}

func (m *MetricRelabel) Mutate(prometheusMetric model.Metric) {
	if prometheusMetric == nil {
		return
	}

	newMetric := Process(prometheusMetric, m.Configs...)

	for k := range prometheusMetric {
		delete(prometheusMetric, k)
	}

	//nothing to do
	if newMetric == nil {
		return
	}

	//assign with new values
	for k, v := range newMetric {
		prometheusMetric[k] = v
	}
}

func (m *MetricRelabel) Name() string {
	return "Prometheus metric relabel"
}

func relabel(metric model.Metric, cfg *Config) model.Metric {
	values := make([]string, 0, len(cfg.SourceLabels))
	for _, ln := range cfg.SourceLabels {
		values = append(values, string(metric[ln]))
	}
	val := strings.Join(values, cfg.Separator)

	mCopy := metric.Clone()

	switch cfg.Action {
	case Drop:
		if cfg.Regex.MatchString(val) {
			return nil
		}
	case Keep:
		if !cfg.Regex.MatchString(val) {
			return nil
		}
	case Replace:
		indexes := cfg.Regex.FindStringSubmatchIndex(val)
		// If there is no match no replacement must take place.
		if indexes == nil {
			break
		}
		target := model.LabelName(cfg.Regex.ExpandString([]byte{}, cfg.TargetLabel, val, indexes))
		if !target.IsValid() {
			delete(mCopy, model.LabelName(cfg.TargetLabel))
			break
		}
		res := cfg.Regex.ExpandString([]byte{}, cfg.Replacement, val, indexes)
		if len(res) == 0 {
			delete(mCopy, model.LabelName(cfg.TargetLabel))
			break
		}
		mCopy[target] = model.LabelValue(res)
	case HashMod:
		mod := sum64(md5.Sum([]byte(val))) % cfg.Modulus
		mCopy[model.LabelName(cfg.TargetLabel)] = model.LabelValue(fmt.Sprintf("%d", mod))
	case LabelMap:
		for k, v := range metric {
			if cfg.Regex.MatchString(string(k)) {
				res := cfg.Regex.ReplaceAllString(string(k), cfg.Replacement)
				mCopy[model.LabelName(res)] = v
			}
		}
	case LabelDrop:
		for k := range metric {
			if cfg.Regex.MatchString(string(k)) {
				delete(mCopy, k)
			}
		}
	case LabelKeep:
		for k := range metric {
			if !cfg.Regex.MatchString(string(k)) {
				delete(mCopy, k)
			}
		}
	default:
		panic(errors.Errorf("relabel: unknown relabel action type %q", cfg.Action))
	}

	return mCopy
}

// sum64 sums the md5 hash to an uint64.
func sum64(hash [md5.Size]byte) uint64 {
	var s uint64

	for i, b := range hash {
		shift := uint64((md5.Size - i - 1) * 8)

		s |= uint64(b) << shift
	}
	return s
}
