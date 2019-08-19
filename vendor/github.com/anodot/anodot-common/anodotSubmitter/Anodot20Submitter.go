package anodotSubmitter

import (
	"net/url"
	"net/http"
	"time"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"fmt"
	"log"
	"github.com/anodot/anodot-common/anodotParser"
	"github.com/anodot/anodot-common/remoteStats"
)

const PATH  = "/api/v1/metrics/"
const CONTENT_TYPE  = "application/json"
const METHOD  = "POST"
const PROTOCOL = "anodot20"

type Anodot20Submitter struct {
	Url       string
	Port      string
	Token     string
	Stats* 	  remoteStats.RemoteStats
	MirrorUrl	string
	MirrorPort	string
	MirrorToken	string
}

type AnodotResponse struct {
	errors []map[string]string `json:"errors"`
}

func NewAnodot20Submitter(url string, port string, token string,
			stats* remoteStats.RemoteStats, murl string,mport string,mtoken string)(s Anodot20Submitter) {
	return Anodot20Submitter{Url:url,Port:port,Token:token,Stats:stats,MirrorUrl:murl,MirrorPort:mport,MirrorToken:mtoken}
}

func (s *Anodot20Submitter) MirrorMetrics(metrics *[]anodotParser.AnodotMetric){
	u, _ := url.ParseRequestURI(s.MirrorUrl+":"+s.MirrorPort)
	u.Path  = PATH
	q := u.Query()
	q.Set("token",s.MirrorToken)
	s.submitMetrics(metrics,u,q)
	s.Stats.UpdateHist(remoteStats.MIRROR_REMOTE_SAMPLES_PER_REQUEST,int64(len(*metrics)))
	s.Stats.UpdateMeter(remoteStats.MIRROR_SUBMITTED_SAMPLES,int64(len(*metrics)))
	s.Stats.UpdateMeter(remoteStats.MIRROR_REMOTE_REQUESTS,1)
}

func (s *Anodot20Submitter) submitMetrics(metrics *[]anodotParser.AnodotMetric,u *url.URL,q url.Values){
	q.Set("protocol",PROTOCOL)
	u.RawQuery = q.Encode()
	urlStr := fmt.Sprintf("%v", u)
	client := &http.Client{Timeout:time.Duration(30 * time.Second)}
	b, e := json.Marshal(*metrics)
	if e != nil{
		log.Printf("Failed to parse message:"+e.Error())
		return
	}
	r, _ := http.NewRequest(METHOD, urlStr,bytes.NewBuffer(b))
	r.Header.Add("Content-Type", CONTENT_TYPE)
	resp,err := client.Do(r)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if resp.StatusCode != 200{

		log.Println("Http Error:",resp.StatusCode)
		s.Stats.UpdateMeter(remoteStats.REMOTE_HTTP_ERRORS,1)
		return
	}

	if resp.Body == nil{
		fmt.Println("Empty response body")
		s.Stats.UpdateMeter(remoteStats.REMOTE_HTTP_ERRORS,1)
		return
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)


	var anodotResponse AnodotResponse
	json.Unmarshal(bodyBytes,anodotResponse)

	if e != nil{
		log.Printf("Failed to parse response:"+e.Error())
		s.Stats.UpdateMeter(remoteStats.REMOTE_HTTP_ERRORS,1)
	}

	if anodotResponse.errors != nil{
		fmt.Println(anodotResponse)
	}

}

func (s *Anodot20Submitter) SubmitMetrics(metrics *[]anodotParser.AnodotMetric)  {
	u, _ := url.ParseRequestURI(s.Url+":"+s.Port)
	u.Path  = PATH
	q := u.Query()
	q.Set("token",s.Token)
	s.submitMetrics(metrics,u,q)
	s.Stats.UpdateHist(remoteStats.REMOTE_SAMPLES_PER_REQUEST,int64(len(*metrics)))
	s.Stats.UpdateMeter(remoteStats.SUBMITTED_SAMPLES,int64(len(*metrics)))
	s.Stats.UpdateMeter(remoteStats.REMOTE_REQUESTS,1)
}