package relabling

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"io/ioutil"
	log "k8s.io/klog/v2"
	"net/http"
	"net/url"
	"time"
)

type PodsMapping struct {
	WhitelistedPods *PodCache
	ExcludedPods    *PodCache

	podRelabelURL url.URL
}

func NewPodsMappingProvider(podRelabelURL string) (*PodsMapping, error) {
	log.V(3).Infof("kubernetes pods relabel enabled. service URL=%q.Max retries=%d", podRelabelURL, 3)
	parsedUrl, err := url.Parse(podRelabelURL)
	if err != nil {
		return nil, err
	}

	parsedUrl.Path = "/pods"

	return &PodsMapping{
		WhitelistedPods: NewCache(),
		ExcludedPods:    NewCache(),
		podRelabelURL:   *parsedUrl,
	}, nil
}

func (p *PodsMapping) UpdateConfig() error {
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{Timeout: time.Second * 10}
	client.RetryMax = 3

	request, err := retryablehttp.NewRequest("GET", p.podRelabelURL.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to get pod mapping configuration: %s ", err.Error())
	}

	if response == nil || response.Body == nil {
		return fmt.Errorf("failed to get pod mapping configuration empty response")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to get pod mapping configuration: %s", err.Error())
	}

	err = response.Body.Close()
	if err != nil {
		return err
	}

	log.V(5).Infof("fetched config: %s", string(body))

	var r PodsMapping
	err = json.Unmarshal(body, &r)
	if err != nil {
		return fmt.Errorf("failed to parse: %s", err.Error())
	}

	p.WhitelistedPods.Replace(r.WhitelistedPods.Data)
	p.ExcludedPods.Replace(r.ExcludedPods.Data)

	return nil
}
