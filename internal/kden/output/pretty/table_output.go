package pretty

import (
	"bytes"
	"fmt"
	"os"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

func FormatTable(modelFunc ModelFunc, modelFuncData interface{}) (string, error) {
	opts := []tea.ProgramOption{}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		var buf bytes.Buffer
		opts = append(opts, tea.WithInput(nil), tea.WithOutput(&buf))
	}

	_, err := tea.NewProgram(NewTableModel(modelFunc, false, modelFuncData), opts...).Run()
	if err != nil {
		return "", fmt.Errorf("error during table creation for output type: %s: %s",
			cfg.Config.Output, err.Error())
	}

	return "", nil
}
