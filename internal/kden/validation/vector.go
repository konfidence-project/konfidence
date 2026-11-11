package validation

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
)

func ValidateVector(constructorData []byte, filePath string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	constructor, err := ocm.ParseComponentConstructor(string(constructorData), filePath)
	if err != nil {
		return output.ExtractSchemaValidationErrors(err, filePath)
	}

	var validationErrors []output.SchemaValidationError
	for componentIndex, component := range constructor.Components {
		seen := make(map[string]string, len(component.References))

		for referenceIndex, reference := range component.References {
			previousVersion, exists := seen[reference.Component]
			if !exists {
				seen[reference.Component] = reference.Version
				continue
			}

			path := fmt.Sprintf("/components/%d/componentReferences/%d/componentName", componentIndex, referenceIndex)
			message := fmt.Sprintf("artifact %q must not appear more than once in a vector; found versions %q and %q",
				reference.Component, previousVersion, reference.Version)
			validationErrors = append(validationErrors, output.ExtractSchemaValidationError(message, path, filePath)...)
		}
	}

	return validationErrors, nil
}
