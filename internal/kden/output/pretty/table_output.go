package pretty

import (
	"fmt"
)

// FormatTable renders a static table for the given command and prints it to
// stdout. The output is rendered directly without a TUI lifecycle.
func FormatTable(modelFunc ModelFunc, modelFuncData interface{}) error {
	data := modelFunc(modelFuncData)
	if data.Err != nil {
		return data.Err
	}

	fmt.Println(renderTable(data))
	return nil
}
