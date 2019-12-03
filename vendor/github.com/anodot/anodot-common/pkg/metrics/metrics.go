package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AnodotTimestamp struct {
	time.Time
}

func (t *AnodotTimestamp) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprint(t.Unix())
	return []byte(stamp), nil
}

type Anodot20Metric struct {
	Properties map[string]string `json:"properties"`
	Timestamp  AnodotTimestamp   `json:"timestamp"`
	Value      float64           `json:"value"`
	Tags       map[string]string `json:"tags"`
}

type Submitter interface {
	SubmitMetrics(metrics []Anodot20Metric) (*AnodotResponse, error)
	AnodotURL() *url.URL
}

//  Anodot 2.0 Metrics submitter.
// See more details at https://support.anodot.com/hc/en-us/articles/360020259354-Posting-2-0-Metrics-
type Anodot20Submitter struct {
	ServerURL *url.URL
	Token     string

	client *http.Client
}

// Anodot server response.
// See more at: https://app.swaggerhub.com/apis/Anodot/metrics_protocol_2.0/1.0.0#/ErrorResponse
type AnodotResponse struct {
	Errors []struct {
		Description string
		Error       int64
		Index       string
	} `json:"errors"`
	HttpResponse *http.Response `json:"-"`
}

func (r *AnodotResponse) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *AnodotResponse) ErrorMessage() string {
	return fmt.Sprintf("%+v\n", r.Errors)
}

var (
	anoServerhttpReponses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anodot_server_http_responses_total",
		Help: "Total number of HTTP responses of Anodot server",
	}, []string{"server", "response_code"})
)

//Constructs new Anodot 2.0 submitter which should be used to send metrics to Anodot.
func NewAnodot20Submitter(anodotURL string, apiToken string, httpClient *http.Client) (*Anodot20Submitter, error) {
	parsedUrl, err := url.Parse(anodotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Anodot server url: %w", err)
	}

	if len(strings.TrimSpace(apiToken)) == 0 {
		return nil, fmt.Errorf("anodot api token should not be blank")
	}

	submitter := Anodot20Submitter{Token: apiToken, ServerURL: parsedUrl, client: httpClient}
	if httpClient == nil {
		//TODO somehow add debug info
		submitter.client = &http.Client{Timeout: 30 * time.Second}
	}

	return &submitter, nil
}

func (s *Anodot20Submitter) SubmitMetrics(metrics []Anodot20Metric) (*AnodotResponse, error) {
	s.ServerURL.Path = "/api/v1/metrics"

	q := s.ServerURL.Query()
	q.Set("token", s.Token)
	q.Set("protocol", "anodot20")

	s.ServerURL.RawQuery = q.Encode()

	b, e := json.Marshal(metrics)
	if e != nil {
		return nil, fmt.Errorf("Failed to parse message:" + e.Error())
	}

	r, _ := http.NewRequest(http.MethodPost, s.ServerURL.String(), bytes.NewBuffer(b))
	r.Header.Add("Content-Type", "application/json")

	resp, err := s.client.Do(r)
	anodotResponse := &AnodotResponse{HttpResponse: resp}
	if err != nil {
		return anodotResponse, err
	}

	anoServerhttpReponses.WithLabelValues(s.AnodotURL().Host, strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode != 200 {
		return anodotResponse, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bodyBytes, anodotResponse)
	if err != nil {
		return anodotResponse, fmt.Errorf("failed to parse Anodot sever response: %w ", err)
	}

	if anodotResponse.HasErrors() {
		return anodotResponse, errors.New(anodotResponse.ErrorMessage())
	} else {
		return anodotResponse, nil
	}
}

func (s *Anodot20Submitter) AnodotURL() *url.URL {
	return s.ServerURL
}
