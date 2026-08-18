package pretty

import (
	"fmt"
	"sync"

	"github.com/konfidence-project/konfidence/internal/kden/validation/output"

	"charm.land/bubbles/v2/table"
)

var (
	modelFuncMap map[string]ModelFunc
	once         sync.Once
)

func GetModelFuncMap() map[string]ModelFunc {
	once.Do(func() {
		modelFuncMap = map[string]ModelFunc{
			"validate": validateModelFunc,
		}
	})
	return modelFuncMap
}

func validateModelFunc(data interface{}) *TableData {
	msg, ok := data.([]output.SchemaValidationError)
	if !ok {
		return &TableData{
			Err: fmt.Errorf("error while creating table for command validate: "+
				"expected []SchemaValidationError, got %T", data),
		}
	}

	columns := []table.Column{
		{Title: "File", Width: 40},
		{Title: "Path", Width: 80},
		{Title: "Message", Width: 80},
	}

	rows := make([]table.Row, 0, len(msg))
	for _, e := range msg {
		rows = append(rows, table.Row{
			e.File,
			e.Path,
			e.Message,
		})
	}

	return &TableData{
		Columns: columns,
		Rows:    rows,
	}
}
