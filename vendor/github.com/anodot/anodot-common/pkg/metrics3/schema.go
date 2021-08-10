package metrics3

type Measurment struct {
	Base MeasurmentBase
	Name string
}

type MeasurmentBase struct {
	Aggregation string `json:"aggregation"`
	CountBy     string `json:"countBy"`
	Units       string `json:"units,omitempty"`
}

type DimensionPolicy struct {
	Action string `json:"action"`
	Fill   string `json:"fill,omitempty"`
}

type AnodotMetricsSchema struct {
	Id               string                    `json:"id,omitempty"`
	Dimensions       []string                  `json:"dimensions"`
	Measurements     map[string]MeasurmentBase `json:"measurements"`
	MissingDimPolicy *DimensionPolicy          `json:"missingDimPolicy,omitempty"`
	Name             string                    `json:"name"`
	Version          string                    `json:"version,omitempty"`
}

type StreamSchemaWrapper struct {
	Wrapper struct {
		Schema AnodotMetricsSchema `json:"schema"`
	} `json:"streamSchemaWrapper"`
}

// Responses for api calls
// Inherits base methods and fields from ApiResponse structure using composition
type GetSchemaResponse struct {
	Schemas []AnodotMetricsSchema
	Api30Response
}

type DeleteSchemaResponse struct {
	SchemaId *string
	Api30Response
}

type CreateSchemaResponse struct {
	SchemaId *string
	Api30Response
}
