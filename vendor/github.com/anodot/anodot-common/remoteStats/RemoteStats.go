package remoteStats

import (
	"github.com/rcrowley/go-metrics"
)

const (
	SUBMITTED_SAMPLES     = "SUBMITTED_SAMPLES"
	PARSING_ERRORS    = "PARSING_ERRORS"
	REMOTE_HTTP_ERRORS = "REMOTE_HTTP_ERRORS"
	CLIENT_REQUESTS    = "CLIENT_REQUESTS"
	REMOTE_REQUESTS     = "REMOTE_REQUESTS"
	CONCURRENT_REQUESTS     = "CONCURRENT_REQUESTS"
	REMOTE_SAMPLES_PER_REQUEST		= "REMOTE_SAMPLES_PER_REQUEST"
	REMOTE_REQUEST_TIME			= "REMOTE_REQUEST_TIME"
	BAD_REQUESTS = "BAD_REQUESTS"
	SERVER_ERROR = "SERVER_ERROR"
	MIRROR_REMOTE_SAMPLES_PER_REQUEST = "MIRROR_REMOTE_SAMPLES_PER_REQUEST"
	MIRROR_SUBMITTED_SAMPLES = "MIRROR_SUBMITTED_SAMPLES"
	MIRROR_REMOTE_REQUESTS = "MIRROR_REMOTE_REQUESTS"
)

type RemoteStats struct{
	Registry metrics.Registry
}

func NewStats() (RemoteStats) {
	var s RemoteStats
	s.Registry = metrics.NewRegistry()
	return s
}

func (s *RemoteStats) UpdateMeter(meter string,mark int64)  {

	 m := metrics.GetOrRegisterMeter(meter,s.Registry)
	 m.Mark(mark)
}

func (s *RemoteStats) UpdateGauge(gauge string,value int64)  {

	g := metrics.GetOrRegisterGauge(gauge,s.Registry)
	g.Update(value)
}

func (s *RemoteStats) UpdateHist(histogram string,value int64)  {

	h := metrics.GetOrRegisterHistogram(histogram,s.Registry,metrics.NewUniformSample(1000))
	h.Update(value)


}




