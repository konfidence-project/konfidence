package validation

import (
	"fmt"
	"os"

	output "github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/maps"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
)

const (
	requiredResourceTypeValue = "cloud.konfidence.artifact.manifest"
	requiredResourcePathKey   = "path"
	requiredFileTypeKey       = "type"

	typeFieldViolationPath = "components/items/properties/resources/$ref/%s/type"
)

func ValidateArtifact(constructorData []byte, filePath string, resourceJsonPaths map[string]bool) ([]output.SchemaValidationError, error) {
	componentConstructor, err := ocm.ParseComponentConstructor(string(constructorData), filePath)
	if err != nil {
		return output.ExtractSchemaValidationErrors(err, filePath)
	}

	if err := ValidateRawYAML(constructorData); err != nil {
		return output.ExtractSchemaValidationErrors(err, filePath)
	}

	for _, component := range componentConstructor.Components {
		for _, resource := range component.Resources {

			if resource.Type != requiredResourceTypeValue {
				continue
			}

			pathVal, err := maps.GetValueFromRawMap(resource.Input.Data, requiredResourcePathKey)
			if err != nil {
				return nil, fmt.Errorf("failed to read 'path' key: %w", err)
			}

			if pathVal == nil {
				return output.ExtractSchemaValidationError("property 'path' is nil", "/path", filePath), nil
			}

			pathStr, ok := pathVal.(string)
			if !ok || pathStr == "" {
				return nil, fmt.Errorf("'path' must be a non-empty string")
			}

			if maps.CheckIfValueIsPresent(resourceJsonPaths, pathStr) {
				continue
			}

			data, err := os.ReadFile(pathStr)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %q: %w", pathStr, err)
			}

			typeVal, err := maps.GetValueFromRawMap(data, requiredFileTypeKey)
			if err != nil {
				return output.ExtractSchemaValidationError("missing property 'type'", getFieldViolationPaths(pathStr), filePath), nil
			}

			if typeVal == nil {
				return output.ExtractSchemaValidationError("property 'type' is nil", getFieldViolationPaths(pathStr), filePath), nil
			}

			typeStr, ok := typeVal.(string)
			if !ok || typeStr == "" {
				return output.ExtractSchemaValidationError("missing or empty value string property 'type'", getFieldViolationPaths(pathStr), filePath), nil
			}

			resourceJsonPaths[pathStr] = true
		}
	}

	return nil, nil
}

func getFieldViolationPaths(filePath string) string {
	return fmt.Sprintf(typeFieldViolationPath, filePath)
}
