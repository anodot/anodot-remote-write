package remoteStats

type MockRemoteStats struct{

}

func (s *MockRemoteStats) UpdateMeter(meter string,mark int64)()  {

}

func (s *MockRemoteStats) UpdateGauge(gauge string,value int64) () {

}

func (s *MockRemoteStats) UpdateHist(histogram string,value int64) () {

}