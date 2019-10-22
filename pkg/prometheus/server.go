package prometheus

import (
	"fmt"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/anodotSubmitter"
	"github.com/anodot/anodot-common/remoteStats"
	"github.com/anodot/anodot-remote-write/pkg/remote"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"io/ioutil"
	"log"
	"net/http"
)

type Receiver struct {
	Port   int
	Parser *anodotParser.AnodotParser
}

const RECEIVER_ENDPOINT = "/receive"
const HEALTH_ENDPOINT = "/health"

func (rc *Receiver) protoToSamples(req *prompb.WriteRequest) model.Samples {
	var samples model.Samples
	for _, ts := range req.Timeseries {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}

		for _, s := range ts.Samples {
			samples = append(samples, &model.Sample{
				Metric:    metric,
				Value:     model.SampleValue(s.Value),
				Timestamp: model.Time(s.Timestamp),
			})
		}
	}
	return samples
}

func (rc *Receiver) InitHttp(s *anodotSubmitter.Anodot20Submitter, stats *remoteStats.RemoteStats, workers *remote.Worker) {

	http.HandleFunc(RECEIVER_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {

		stats.UpdateMeter(remoteStats.CLIENT_REQUESTS, 1)

		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			stats.UpdateMeter(remoteStats.SERVER_ERROR, 1)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			stats.UpdateMeter(remoteStats.BAD_REQUESTS, 1)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			stats.UpdateMeter(remoteStats.BAD_REQUESTS, 1)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		metrics := rc.Parser.ParsePrometheusRequest(rc.protoToSamples(&req), stats)
		workers.Do(metrics, s)
	})

	http.HandleFunc(HEALTH_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", rc.Port), nil))

}
