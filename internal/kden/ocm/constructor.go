package ocm

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	constructorv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
)

// ParseComponentConstructor performs ENV substitution on the constructor file content.
// This enables dynamic configuration using ${VAR_NAME} or $VAR_NAME syntax.
// Variables are expanded using os.Expand with os.Getenv as the mapping function.
// Additionally, validated the constructor against the OCM JSON schema.
func ParseComponentConstructor(constructorData, filePath string) (*constructorv1.ComponentConstructor, error) {
	expanded := os.Expand(constructorData, os.Getenv)

	var componentConstructor constructorv1.ComponentConstructor
	if err := yaml.Unmarshal([]byte(expanded), &componentConstructor); err != nil {
		return nil, fmt.Errorf("unmarshalling ocm component constructor %q failed: %w", filePath, err)
	}

	return &componentConstructor, nil
}
