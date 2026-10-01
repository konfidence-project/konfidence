package pretty

import "fmt"

// RenderTable renders generic table data as text.
func RenderTable(data *TableData) (string, error) {
	if data == nil {
		return "", fmt.Errorf("pretty output requires table data")
	}

	return renderTable(data), nil
}
