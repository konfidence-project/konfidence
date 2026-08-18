package validation

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/output"
	voutput "github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/fs"
	"github.com/konfidence-project/konfidence/pkg/maps"
)

// ValidateConfig holds the parameters needed to run a validate command.
type ValidateConfig struct {
	CmdDisplayName      string
	DefaultFile         string
	ComponentIdentifier string
	ValidateFn          func([]byte, string, map[string]bool) ([]voutput.SchemaValidationError, error)
}

type SchemaValidationError struct {
	ErrorMessage string
}

func (s SchemaValidationError) Error() string {
	return s.ErrorMessage
}

// RunValidate executes the common validate loop for any resource type.
func RunValidate(filePaths []string, cfg ValidateConfig) error {
	if len(filePaths) == 0 {
		filePaths = []string{cfg.DefaultFile}
	}
	filePaths = maps.GetDistinctValues(filePaths)

	files, err := fs.ToFileData(filePaths)
	if err != nil {
		return err
	}

	resourceJsonPaths := map[string]bool{}
	var schemaValidationErrors []voutput.SchemaValidationError
	for _, f := range files {
		data, err := fs.ReadFile(f)
		if err != nil {
			return err
		}

		errs, err := cfg.ValidateFn(data, f.GetFilePath(), resourceJsonPaths)
		if err != nil {
			return fmt.Errorf("failed to validate %s %s: %w", cfg.ComponentIdentifier, f.GetFilePath(), err)
		}
		schemaValidationErrors = append(schemaValidationErrors, errs...)
	}

	if len(schemaValidationErrors) > 0 {
		result, err := output.ResolveFormat(schemaValidationErrors, cfg.CmdDisplayName)
		if err != nil {
			return fmt.Errorf("failed to resolve output format for validate command: %w", err)
		}
		return &SchemaValidationError{ErrorMessage: result}
	}

	return nil
}
