package validation

import (
	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
)

func ValidateVector(constructorData []byte, filePath string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	_, err := ocm.ParseComponentConstructor(string(constructorData), filePath)
	if err != nil {
		return output.ExtractSchemaValidationErrors(err, filePath)
	}
	return nil, nil
}
