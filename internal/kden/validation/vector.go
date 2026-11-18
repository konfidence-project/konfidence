package validation

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/collection"
	constructorv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
)

func ValidateVector(constructorData []byte, filePath string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	constructor, err := ocm.ParseComponentConstructor(string(constructorData), filePath)
	if err != nil {
		return output.ExtractSchemaValidationErrors(err, filePath)
	}

	var validationErrors []output.SchemaValidationError
	for componentIndex, component := range constructor.Components {
		duplicates := collection.FindDuplicatesBy(component.References, func(reference constructorv1.Reference) string {
			return reference.Component
		})

		for _, duplicate := range duplicates {
			path := fmt.Sprintf("/components/%d/componentReferences/%d/componentName", componentIndex, duplicate.DuplicateIndex)
			message := fmt.Sprintf("artifact %q must not appear more than once in a vector; found versions %q and %q",
				duplicate.Duplicate.Component,
				duplicate.First.Version,
				duplicate.Duplicate.Version,
			)
			validationErrors = append(validationErrors, output.ExtractSchemaValidationError(message, path, filePath)...)
		}
	}

	return validationErrors, nil
}
