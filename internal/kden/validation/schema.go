package validation

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

//go:embed resources/konfidence-artifact-schema.json
var konfidenceSchema []byte

const schemaID = "cloud.konfidence/konfidence-artifact-schema.json"

var getSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()

	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(konfidenceSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy schema: %w", err)
	}
	if err := c.AddResource(schemaID, schema); err != nil {
		return nil, fmt.Errorf("failed to add policy schema resource: %w", err)
	}

	return c.Compile(schemaID)
})

func ValidateRawJSON(raw []byte) error {
	var mm map[string]any
	if err := json.Unmarshal(raw, &mm); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	s, err := getSchema()
	if err != nil {
		return err
	}
	return s.Validate(mm)
}

func ValidateRawYAML(raw []byte) error {
	var mm map[string]any
	if err := yaml.Unmarshal(raw, &mm); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	s, err := getSchema()
	if err != nil {
		return err
	}
	return s.Validate(mm)
}
