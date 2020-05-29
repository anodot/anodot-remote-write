package remote

import (
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"io"
	"io/ioutil"
	log "k8s.io/klog/v2"
	"net/http"
	"os"
	"strings"
	"time"
)

func FetchMetrics(url string, retries int, timeout time.Duration) ([]*model.Sample, error) {
	var (
		samples  []*model.Sample
		err      error
		response *http.Response
	)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	var scrapeTime model.Time
	for retries > 0 {
		response, err = client.Get(url)
		if err != nil {
			retries -= 1
			time.Sleep(timeout)
		} else {
			scrapeTime = model.Now()
			break
		}
	}

	if err != nil {
		return samples, err
	}

	defer func() {
		_, err := io.Copy(ioutil.Discard, response.Body)
		log.Error(err)
		_ = response.Body.Close()
	}()

	dec := expfmt.NewDecoder(response.Body, expfmt.FmtText)
	decoder := expfmt.SampleDecoder{
		Dec:  dec,
		Opts: &expfmt.DecodeOptions{},
	}

	for {
		var v model.Vector
		if err := decoder.Decode(&v); err != nil {
			if err == io.EOF {
				for _, s := range samples {
					s.Timestamp = scrapeTime

					instanceName := os.Getenv("ANODOT_INSTANCE_NAME")
					if len(strings.TrimSpace(instanceName)) > 0 {
						s.Metric[model.LabelName("anodot_tag_source_host_id")] = model.LabelValue(instanceName)
					}
				}
				return samples, nil
			}
			return nil, err
		}
		samples = append(samples, v...)
	}
}
