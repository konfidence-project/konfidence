package pretty

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	newLineSeparator = "\n"
	QuitKey          = "q"
	QuitCombination  = "ctrl+c"

	headerHeight = 1
	maxHeight    = 500
	maxWidth     = 500
)

var quitMsgKeys = []string{QuitKey, QuitCombination}

type TableModel struct {
	table         table.Model
	loading       bool
	modelFunc     ModelFunc
	modelFuncData interface{}
	spinner       spinner.Model
	baseStyle     lipgloss.Style
	interactive   bool
}

type TableData struct {
	Columns []table.Column
	Rows    []table.Row
	Err     error
}

// ModelFunc defines a dynamic data loader for tables
type ModelFunc func(data interface{}) *TableData

func NewTableModel(modelFunc ModelFunc, interactive bool, modelFuncData interface{}) *TableModel {
	return &TableModel{
		table:         table.New(),
		loading:       true,
		modelFunc:     modelFunc,
		modelFuncData: modelFuncData,
		spinner:       spinner.New(),
		baseStyle:     lipgloss.NewStyle(),
		interactive:   interactive,
	}
}

func (t *TableModel) Init() tea.Cmd {
	if t.modelFunc != nil {
		return tea.Batch(t.spinner.Tick, func() tea.Msg {
			return t.modelFunc(t.modelFuncData)
		})
	}
	return nil
}

func (t *TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case *TableData:
		if msg.Err != nil {
			t.loading = false
			tea.Printf("error while fetching data: %v\n", msg.Err)
			return t, tea.Quit
		}
		t.table = newTable(msg.Columns, msg.Rows)
		t.loading = false

		if t.interactive {
			return t, nil
		}

		return t, tea.Quit
	case tea.KeyMsg:
		if key := msg.String(); slices.Contains(quitMsgKeys, key) {
			return t, tea.Quit
		}
	case spinner.TickMsg:
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}

	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

func newTable(columns []table.Column, rows []table.Row) table.Model {
	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle()

	return table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(getHeight(rows)),
		table.WithWidth(maxWidth),
		table.WithStyles(styles),
	)
}

func getHeight(rows []table.Row) int {
	count := len(rows) + headerHeight
	if count > maxHeight {
		return maxHeight
	}
	return count
}

func (t *TableModel) View() tea.View {
	if t.loading {
		return tea.NewView(fmt.Sprintf("%s Loading...%s", t.spinner.View(), newLineSeparator))
	}

	view := fmt.Sprintf("%s %s %s", t.table.View(), newLineSeparator, t.table.HelpView())
	return tea.NewView(t.baseStyle.Render(view))
}
