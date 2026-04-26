package rapidex

// MappingConfig defines the schema for mapping RapidPro flow variables to
// DHIS2 aggregates.  The YAML keys mirror those in Rapidex v1.  Each
// mapping configuration is tied to a specific RapidPro flow UUID.
//
// Example YAML:
//
//	flow_uuid: "123e4567-e89b-12d3-a456-426614174000"
//	flow_name: "Weekly Report"
//	dataset: "DATASET_UID"
//	org_unit_var: "facility_code"
//	period_var: "reporting_week"
//	payload_aoc: "AOC_UID"
//	mappings:
//	  - field: "blood_pressure"
//	    data_element: "DE1"
//	    category_option_combo: "COC1"
//	    attribute_option_combo: "AOC1"
//	  - field: "heart_rate"
//	    data_element: "DE2"
//	    category_option_combo: "COC2"
//
// See docs/rapidex-v2-overview.md for further guidance.
type MappingConfig struct {
	FlowUUID   string             `yaml:"flow_uuid" json:"flowUuid"`
	FlowName   string             `yaml:"flow_name" json:"flowName"`
	Dataset    string             `yaml:"dataset" json:"dataset"`
	OrgUnitVar string             `yaml:"org_unit_var" json:"orgUnitVar"`
	PeriodVar  string             `yaml:"period_var" json:"periodVar"`
	PayloadAOC string             `yaml:"payload_aoc" json:"payloadAoc"`
	Mappings   []DataValueMapping `yaml:"mappings" json:"mappings"`
}

// DataValueMapping binds a RapidPro variable (`field`) to a DHIS2 data element
// and optional category/attribute option combos.  At least the data element
// must be provided.  The value extracted from the RapidPro webhook will be
// converted to a string and used as the `value` in the aggregate payload.
type DataValueMapping struct {
	Field                string `yaml:"field" json:"field"`
	DataElement          string `yaml:"data_element" json:"dataElement"`
	CategoryOptionCombo  string `yaml:"category_option_combo" json:"categoryOptionCombo"`
	AttributeOptionCombo string `yaml:"attribute_option_combo" json:"attributeOptionCombo"`
}

// AggregatePayload mirrors the structure expected by DHIS2 for an
// aggregate dataSet submission.  It is intentionally simple and does
// not include optional metadata such as `completeDate`.
type AggregatePayload struct {
	DataSet              string      `json:"dataSet"`
	OrgUnit              string      `json:"orgUnit"`
	Period               string      `json:"period"`
	AttributeOptionCombo string      `json:"attributeOptionCombo,omitempty"`
	DataValues           []DataValue `json:"dataValues"`
}

// DataValue represents a single data element/value pair in the aggregate
// payload.  Category and attribute option combos are optional and may be
// inherited from the payload level.
type DataValue struct {
	DataElement          string `json:"dataElement"`
	CategoryOptionCombo  string `json:"categoryOptionCombo,omitempty"`
	AttributeOptionCombo string `json:"attributeOptionCombo,omitempty"`
	Value                string `json:"value"`
}

// RapidProWebhook is a simplified representation of the webhook body
// delivered by RapidPro when a flow completes.  Only the fields required
// for mapping are included here.  Additional properties (such as contact
// fields) may be stored in the Maps.
type RapidProWebhook struct {
	FlowUUID string                 `json:"flow_uuid"`
	Results  map[string]interface{} `json:"results"`
	Fields   map[string]interface{} `json:"fields"`
	Contact  struct {
		UUID   string                 `json:"uuid"`
		Fields map[string]interface{} `json:"fields"`
	} `json:"contact"`
	Extra map[string]interface{} `json:"extra"`
}
