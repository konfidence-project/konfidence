package pretty

import (
	"fmt"
)

// FormatTable renders a static table for the given command and prints it to
// stdout. The output is one-shot (no scrolling, no interaction), so it renders
// the view directly instead of running a bubbletea program — that avoids the
// terminal-probe escape sequences and scroll-help line a full TUI would emit for
// what is really a plain table.
func FormatTable(modelFunc ModelFunc, modelFuncData interface{}) error {
	data := modelFunc(modelFuncData)
	if data.Err != nil {
		return data.Err
	}

	fmt.Println(renderTable(data))
	return nil
}
