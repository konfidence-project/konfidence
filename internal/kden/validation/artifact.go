package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	output "github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/maps"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
)

const (
	requiredResourceTypeValue = "cloud.konfidence.artifact.manifest"
	requiredResourcePathKey   = "path"
	requiredFileTypeKey       = "type"
	utf8InputName             = "utf8"

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

			if strings.EqualFold(resource.Input.Name, utf8InputName) {
				data, err := inlineManifest(resource.Input.Data)
				if err != nil {
					return nil, err
				}
				if errs := validateManifestType(data, getFieldViolationPaths(resource.Name), filePath); errs != nil {
					return errs, nil
				}
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

			if errs := validateManifestType(data, getFieldViolationPaths(pathStr), filePath); errs != nil {
				return errs, nil
			}

			resourceJsonPaths[pathStr] = true
		}
	}

	return nil, nil
}

func validateManifestType(data []byte, violationPath, filePath string) []output.SchemaValidationError {
	typeVal, err := maps.GetValueFromRawMap(data, requiredFileTypeKey)
	if err != nil {
		return output.ExtractSchemaValidationError("missing property 'type'", violationPath, filePath)
	}

	if typeVal == nil {
		return output.ExtractSchemaValidationError("property 'type' is nil", violationPath, filePath)
	}

	typeStr, ok := typeVal.(string)
	if !ok || typeStr == "" {
		return output.ExtractSchemaValidationError("missing or empty value string property 'type'", violationPath, filePath)
	}

	return nil
}

// The constructor input is already JSON here, inline yaml included.
func inlineManifest(input []byte) ([]byte, error) {
	var in struct {
		YAML          json.RawMessage `json:"yaml"`
		JSON          json.RawMessage `json:"json"`
		FormattedJSON json.RawMessage `json:"formattedJson"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("failed to read UTF8/v1 input: %w", err)
	}
	for _, content := range [][]byte{in.YAML, in.JSON, in.FormattedJSON} {
		if len(content) > 0 {
			return content, nil
		}
	}
	return nil, nil
}

func getFieldViolationPaths(filePath string) string {
	return fmt.Sprintf(typeFieldViolationPath, filePath)
}
