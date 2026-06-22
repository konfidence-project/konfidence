package schema

//go:generate go run generation.go

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

func MarshalJSONSchema() ([]byte, error) {
	r := &jsonschema.Reflector{
		ExpandedStruct:            true,
		AllowAdditionalProperties: true,
		Namer: func(t reflect.Type) string {
			name := t.Name()
			if len(name) == 0 {
				return name
			}
			return strings.ToLower(name[:1]) + name[1:]
		},
	}
	s := r.Reflect(&ArtifactConstructor{})
	s.ID = SchemaID

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	return append(data, '\n'), nil
}

func WriteSchema(path string) error {
	data, err := MarshalJSONSchema()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
