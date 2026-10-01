package validation

import (
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	voutput "github.com/konfidence-project/konfidence/internal/kden/validation/output"
)

func schemaValidationErrorsTable(errors []voutput.SchemaValidationError) *pretty.TableData {
	builder := pretty.NewTable().Columns("File", "Path", "Message")
	for _, e := range errors {
		builder.Row().
			String("File", e.File).
			String("Path", e.Path).
			String("Message", e.Message)
	}
	return builder.TableData()
}
