package output

import (
	"errors"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type SchemaValidationError struct {
	File    string `json:"file" yaml:"file"`
	Path    string `json:"path" yaml:"path"`
	Message string `json:"output" yaml:"output"`
}

func ExtractSchemaValidationErrors(err error, file string) ([]SchemaValidationError, error) {
	var validationErr *jsonschema.ValidationError
	var out *jsonschema.OutputUnit
	if errors.As(err, &validationErr) {
		out = validationErr.DetailedOutput()
	} else {
		return nil, err
	}

	msgs := extract(out)
	for i := range msgs {
		msgs[i].File = file
	}
	return msgs, nil
}

func extract(out *jsonschema.OutputUnit) []SchemaValidationError {
	var result []SchemaValidationError
	if out.Error != nil {
		result = append(result, SchemaValidationError{
			Path:    out.KeywordLocation,
			Message: out.Error.String(),
		})
	}

	for _, e := range out.Errors {
		result = append(result, extract(&e)...)
	}

	return result
}

func ExtractSchemaValidationError(msg, path, file string) []SchemaValidationError {
	return []SchemaValidationError{{
		File:    file,
		Path:    path,
		Message: msg,
	},
	}
}
