package remoteStats

type RemoteStatsInterface interface {
	UpdateMeter(meter string,mark int64) ()
	UpdateGauge(gauge string,value int64)()
	UpdateHist(histogram string,value int64) ()
}



