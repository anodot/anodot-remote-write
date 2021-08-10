package metrics3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

type AnodotResponse interface {
	HasErrors() bool
	ErrorMessage() string
	RawResponse() *http.Response
}

type Api30Response struct {
	Error *struct {
		Status        int    `json:"status"`
		Name          string `json:"name"`
		Message       string `json:"message"`
		AndtErrorCode int    `json:"andtErrorCode"`
		Path          string `json:"path"`
	}
	HttpResponse *http.Response `json:"-"`
}

func (r *Api30Response) HasErrors() bool {
	return r.Error != nil
}

func (r *Api30Response) ErrorMessage() string {
	return fmt.Sprintf("%+v\n", r.Error)
}

func (r *Api30Response) RawResponse() *http.Response {
	return r.HttpResponse
}

type refreshBearerResponse struct {
	refreshTime time.Time
	bearer      string
	Api30Response
}

type Anodot30Client struct {
	ServerURL           *url.URL
	AccessKey           *string
	DataCollectionToken *string
	client              *http.Client
	bearerToken         *struct {
		timestemp time.Time
		token     string
	}
}

func NewAnodot30Client(anodotURL url.URL, accessKey *string, dataToken *string, httpClient *http.Client) (*Anodot30Client, error) {
	if accessKey == nil && dataToken == nil {
		return nil, fmt.Errorf("anodot token can't be nil")
	}

	submitter := Anodot30Client{AccessKey: accessKey, DataCollectionToken: dataToken, ServerURL: &anodotURL, client: httpClient, bearerToken: nil}
	if httpClient == nil {
		client := http.Client{Timeout: 30 * time.Second}

		debugHTTP, _ := strconv.ParseBool(os.Getenv("ANODOT_HTTP_DEBUG_ENABLED"))
		if debugHTTP {
			client.Transport = &debugHTTPTransport{r: http.DefaultTransport}
		}
		submitter.client = &client
	}

	return &submitter, nil
}

func (c *Anodot30Client) GetBearerToken() (*string, error) {
	// Token valid 24 hours, so if BearerToken field is null or token expired
	// needs to refresh it, otherwise, returns existed token

	if c.bearerToken == nil || time.Since(c.bearerToken.timestemp) > 24*time.Hour {
		resp, err := c.refreshBearerToken()
		if err != nil {
			return nil, err
		}

		if resp.HasErrors() {
			return nil, fmt.Errorf("failed to refresh toke: %v", resp.ErrorMessage())
		}

		c.bearerToken = &struct {
			timestemp time.Time
			token     string
		}{resp.refreshTime, resp.bearer}

	}
	return &c.bearerToken.token, nil
}

func (c *Anodot30Client) refreshBearerToken() (*refreshBearerResponse, error) {

	if c.AccessKey == nil {
		return nil, fmt.Errorf("please provide AccesKey for obtain bearer token")
	}
	sUrl := *c.ServerURL
	sUrl.Path = "api/v2/access-token"

	q := sUrl.Query()
	q.Set("responseformat", "JSON")

	sUrl.RawQuery = q.Encode()

	b, _ := json.Marshal(
		struct {
			RefreshToken string `json:"refreshToken"`
		}{
			*c.AccessKey,
		},
	)

	r, _ := http.NewRequest(http.MethodPost, sUrl.String(), bytes.NewBuffer(b))
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	refreshResponse := refreshBearerResponse{}
	refreshResponse.HttpResponse = resp

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &refreshResponse.Error)
		if err != nil {
			return &refreshResponse,
				fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
		}
		return &refreshResponse, nil
	}

	responseJson := struct{ Token string }{}

	err = json.Unmarshal(bodyBytes, &responseJson)
	if err != nil {
		return &refreshResponse,
			fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
	}

	refreshResponse.bearer = responseJson.Token
	refreshResponse.refreshTime = time.Now()
	return &refreshResponse, nil
}

func (c *Anodot30Client) SubmitMetrics(metrics []AnodotMetrics30) (*SubmitMetricsResponse, error) {
	if c.DataCollectionToken == nil {
		return nil,
			fmt.Errorf("DataCollectionToken should be provided for metrics submit ")
	}

	sUrl := *c.ServerURL
	sUrl.Path = "api/v1/metrics"

	q := sUrl.Query()
	q.Set("token", *c.DataCollectionToken)
	q.Set("protocol", "anodot30")
	sUrl.RawQuery = q.Encode()

	b, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse schema:" + err.Error())
	}
	r, _ := http.NewRequest(http.MethodPost, sUrl.String(), bytes.NewBuffer(b))
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	anodotResponse := &SubmitMetricsResponse{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse)
		if err != nil {
			return anodotResponse, fmt.Errorf("http response is differ from 2xx\nfalied to parse response body: %v \n%s", err, string(bodyBytes))
		}
	}

	err = json.Unmarshal(bodyBytes, anodotResponse)
	if err != nil {
		return anodotResponse,
			fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
	}
	return anodotResponse, nil
}

func (c *Anodot30Client) CreateSchema(schema AnodotMetricsSchema) (*CreateSchemaResponse, error) {
	token, err := c.GetBearerToken()
	if err != nil {
		return nil, err
	}

	var bearer = "Bearer " + *token
	sUrl := c.ServerURL
	sUrl.Path = "/api/v2/stream-schemas"

	b, e := json.Marshal(schema)
	if e != nil {
		return nil,
			fmt.Errorf("Failed to parse schema:" + e.Error())
	}

	r, _ := http.NewRequest(http.MethodPost, sUrl.String(), bytes.NewBuffer(b))

	r.Header.Set("Authorization", bearer)
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	anodotResponse := &CreateSchemaResponse{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse.Error)
		if err != nil {
			return anodotResponse,
				fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
		}
		return anodotResponse, nil
	}

	schemaCreated := struct {
		Schema AnodotMetricsSchema `json:"schema"`
	}{
		Schema: AnodotMetricsSchema{},
	}

	err = json.Unmarshal(bodyBytes, &schemaCreated)
	if err != nil {
		return anodotResponse, err
	}

	anodotResponse.SchemaId = &schemaCreated.Schema.Id
	return anodotResponse, nil
}

func (c *Anodot30Client) DeleteSchema(schemaId string) (*DeleteSchemaResponse, error) {
	token, err := c.GetBearerToken()
	if err != nil {
		return nil, err
	}

	var bearer = "Bearer " + *token
	sUrl := c.ServerURL
	sUrl.Path = "api/v2/stream-schemas/" + schemaId

	r, _ := http.NewRequest(http.MethodDelete, sUrl.String(), nil)

	r.Header.Set("Authorization", bearer)
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	anodotResponse := &DeleteSchemaResponse{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse.Error)
		fmt.Println(anodotResponse.Error)
		if err != nil {
			return anodotResponse,
				fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
		}
		return anodotResponse, nil
	}

	schemaDeleted := struct {
		Deleted string `json:"deleted"`
	}{}

	err = json.Unmarshal(bodyBytes, &schemaDeleted)
	if err != nil {
		return anodotResponse, err
	}

	anodotResponse.SchemaId = &schemaDeleted.Deleted
	return anodotResponse, nil
}

func (c *Anodot30Client) GetSchemas() (*GetSchemaResponse, error) {

	token, err := c.GetBearerToken()
	if err != nil {
		return nil, err
	}

	var bearer = "Bearer " + *token

	sUrl := c.ServerURL
	sUrl.Path = "/api/v2/stream-schemas/schemas"

	r, _ := http.NewRequest(http.MethodGet, sUrl.String(), nil)

	r.Header.Set("Authorization", bearer)

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	anodotResponse := &GetSchemaResponse{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.Body == nil {
		return nil, fmt.Errorf("empty response body")
	}

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse.Error)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
		}
		return anodotResponse, nil
	}

	schemasTmp := make([]StreamSchemaWrapper, 0)
	schemas := make([]AnodotMetricsSchema, 0)

	err = json.Unmarshal(bodyBytes, &schemasTmp)
	if err != nil {
		return anodotResponse, err
	}

	for _, s := range schemasTmp {
		schemas = append(schemas, s.Wrapper.Schema)
	}

	anodotResponse.Schemas = schemas

	return anodotResponse, nil
}

func (c *Anodot30Client) SubmitWatermark(schemaId string, watermark AnodotTimestamp) (*SubmitWatermarkResponse, error) {
	if c.DataCollectionToken == nil {
		return nil,
			fmt.Errorf("DataCollectionToken should be provided for watermark submit ")
	}

	sUrl := *c.ServerURL
	sUrl.Path = "api/v1/metrics/watermark"

	q := sUrl.Query()
	q.Set("token", *c.DataCollectionToken)
	q.Set("protocol", "anodot30")
	sUrl.RawQuery = q.Encode()

	b, _ := json.Marshal(
		struct {
			SchemaId  string          `json:"schemaId"`
			Watermark AnodotTimestamp `json:"watermark"`
		}{schemaId, watermark},
	)
	r, _ := http.NewRequest(http.MethodPost, sUrl.String(), bytes.NewBuffer(b))
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	anodotResponse := SubmitWatermarkResponse{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return &anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse)
		if err != nil {
			return &anodotResponse, fmt.Errorf("http response is differ from 2xx\nfalied to parse response body: %v \n%s", err, string(bodyBytes))
		}
	}

	err = json.Unmarshal(bodyBytes, &anodotResponse)
	if err != nil {
		return &anodotResponse,
			fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
	}

	return &anodotResponse, nil
}

func (c *Anodot30Client) SendToBC(bcData Pipeline) (*Api30Response, error) {
	token, err := c.GetBearerToken()
	if err != nil {
		return nil, err
	}

	var bearer = "Bearer " + *token
	sUrl := *c.ServerURL
	sUrl.Path = "api/v2/bc/agents"

	b, e := json.Marshal(bcData)
	if e != nil {
		return nil, fmt.Errorf("Failed to parse bc data:" + e.Error())
	}

	r, _ := http.NewRequest(http.MethodPost, sUrl.String(), bytes.NewBuffer(b))

	r.Header.Set("Authorization", bearer)
	r.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	anodotResponse := &Api30Response{}
	anodotResponse.HttpResponse = resp

	if resp.Body == nil {
		return anodotResponse, fmt.Errorf("empty response body")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		err = json.Unmarshal(bodyBytes, &anodotResponse.Error)
		if err != nil {
			return anodotResponse, fmt.Errorf("failed to parse reponse body: %v \n%s", err, string(bodyBytes))
		}
		return anodotResponse, nil
	}

	return anodotResponse, nil
}

type debugHTTPTransport struct {
	r http.RoundTripper
}

func (d *debugHTTPTransport) RoundTrip(h *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(h, true)
	fmt.Printf("----------------------------------REQUEST----------------------------------\n%s\n", string(dump))
	resp, err := d.r.RoundTrip(h)
	if err != nil {
		fmt.Println("failed to obtain response: ", err.Error())
		return resp, err
	}

	dump, _ = httputil.DumpResponse(resp, true)
	fmt.Printf("----------------------------------RESPONSE----------------------------------\n%s\n----------------------------------\n\n", string(dump))
	return resp, err
}
