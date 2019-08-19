package anodotSubmitter

import (
	"log"
	"errors"
	"encoding/json"
	"github.com/anodot/anodot-common/anodotParser"
)
type MockSubmitter struct {

}

func (s *MockSubmitter) SubmitMetrics(metrics *[]anodotParser.AnodotMetric)(error)  {

	_, e := json.Marshal(*metrics)
	if e != nil{
		log.Printf("Failed to parse message:"+e.Error())
		return errors.New("Failed to parse message")
	}
	return nil
}
