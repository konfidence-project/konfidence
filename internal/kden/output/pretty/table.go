package pretty

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

const (
	newLineSeparator = "\n"
	headerHeight     = 1

	// cellPadding is the horizontal padding bubbles' table adds per column (one
	// space each side). The table width must account for it or the rightmost
	// column is clipped.
	cellPadding = 2
)

type TableData struct {
	Columns []table.Column
	Rows    []table.Row
	// Sections, when non-empty, is rendered as separate headed tables instead of
	// the flat Columns/Rows above.
	Sections []TableSection
	// Footer is an optional line rendered under the table (e.g. a hint). It is
	// styled dim so it reads as secondary to the data.
	Footer string
	Err    error
}

type TableSection struct {
	Heading string
	Columns []table.Column
	Rows    []table.Row
}

// ModelFunc maps command data to the table shape to render.
type ModelFunc func(data interface{}) *TableData

// renderTable renders a static table (plus optional faint footer) to a string.
func renderTable(data *TableData) string {
	var view string
	if len(data.Sections) > 0 {
		view = renderSections(data.Sections)
	} else {
		view = newTable(data.Columns, data.Rows).View()
	}
	if data.Footer != "" {
		footer := lipgloss.NewStyle().Faint(true).Render(data.Footer)
		view = fmt.Sprintf("%s%s%s", view, newLineSeparator, footer)
	}
	return view
}

func renderSections(sections []TableSection) string {
	parts := make([]string, 0, len(sections))
	for _, s := range sections {
		var b strings.Builder
		if s.Heading != "" {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render(s.Heading))
			b.WriteString(newLineSeparator)
		}
		b.WriteString(newTable(s.Columns, s.Rows).View())
		parts = append(parts, b.String())
	}
	return strings.Join(parts, newLineSeparator+newLineSeparator)
}

func newTable(columns []table.Column, rows []table.Row) table.Model {
	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle()

	return table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+headerHeight),
		table.WithWidth(tableWidth(columns)),
		table.WithStyles(styles),
	)
}

// tableWidth sizes the table to its columns. bubbles' table only renders rows
// when a width is set, so we compute it from the column widths rather than
// forcing a fixed maximum that pads every row far past its content.
func tableWidth(columns []table.Column) int {
	width := 0
	for _, c := range columns {
		width += c.Width + cellPadding
	}
	return width
}
