package jsonschema

import "encoding/json"

const VectorConfigurationV1SchemaVersion = "v1"

// VectorConfigurationV1 is the v1 JSON contract for vector configuration.
type VectorConfigurationV1 struct {
	// SchemaVersion identifies which contract version the payload follows.
	SchemaVersion string `json:"schemaVersion"`

	// Features contains feature flags that influence vector behavior, keyed by
	// their feature name.
	Features json.RawMessage `json:"features,omitempty"`

	// Authored contains user-authored configuration values for the vector.
	Authored json.RawMessage `json:"authored,omitempty"`
}

// NewVectorConfigurationV1 creates a v1 payload and pins the schema version.
func NewVectorConfigurationV1(features, authored json.RawMessage) VectorConfigurationV1 {
	return VectorConfigurationV1{
		SchemaVersion: VectorConfigurationV1SchemaVersion,
		Features:      features,
		Authored:      authored,
	}
}
