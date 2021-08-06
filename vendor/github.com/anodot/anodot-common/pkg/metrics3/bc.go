package metrics3

type Source struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Scheduling struct {
	Interval string `json:"interval"`
	Delay    string `json:"delay"`
}

type Progress struct {
	LastOffset string `json:"last_offset"`
}

type Pipeline struct {
	Id                  string          `json:"pipeline_id"`
	Created             AnodotTimestamp `json:"created"`
	Updated             AnodotTimestamp `json:"updated"`
	Status              string          `json:"status"`
	SchemaId            string          `json:"schemaId"`
	Source              `json:"source"`
	Scheduling          `json:"scheduling"`
	Progress            `json:"progress"`
	AnodotMetricsSchema `json:"schema"`
}
