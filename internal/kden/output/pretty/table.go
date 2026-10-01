package pretty

import (
	"fmt"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

const newLineSeparator = "\n"

type Column struct {
	Title string
	Width int
}

type Row []string

type TableData struct {
	Columns []Column
	Rows    []Row

	// Sections, when non-empty, is rendered as separate headed tables instead
	// of the flat Columns and Rows.
	Sections []TableSection

	Footer string
}

type TableSection struct {
	Heading string
	Columns []Column
	Rows    []Row
}

func renderTable(data *TableData) string {
	var view string
	if len(data.Sections) > 0 {
		view = renderSections(data.Sections)
	} else {
		view = renderRows(data.Columns, data.Rows)
	}
	if data.Footer != "" {
		footer := text.Colors{text.Faint}.Sprint(data.Footer)
		view = fmt.Sprintf("%s%s%s", view, newLineSeparator, footer)
	}

	return view
}

func renderSections(sections []TableSection) string {
	parts := make([]string, 0, len(sections))
	for _, s := range sections {
		var b strings.Builder
		if s.Heading != "" {
			b.WriteString(text.Colors{text.Bold}.Sprint(s.Heading))
			b.WriteString(newLineSeparator)
		}
		b.WriteString(renderRows(s.Columns, s.Rows))
		parts = append(parts, b.String())
	}

	return strings.Join(parts, newLineSeparator+newLineSeparator)
}

func renderRows(columns []Column, rows []Row) string {
	writer := prettytable.NewWriter()
	header := make(prettytable.Row, len(columns))
	columnConfigs := make([]prettytable.ColumnConfig, len(columns))

	for index, column := range columns {
		header[index] = column.Title
		columnConfigs[index] = prettytable.ColumnConfig{
			Number:   index + 1,
			WidthMax: column.Width,
		}
	}

	writer.AppendHeader(header)

	for _, row := range rows {
		renderedRow := make(prettytable.Row, len(row))
		for index, value := range row {
			renderedRow[index] = value
		}
		writer.AppendRow(renderedRow)
	}

	style := prettytable.StyleDefault
	style.Options = prettytable.OptionsNoBordersAndSeparators
	style.Format.Header = text.FormatDefault
	style.Color.Header = text.Colors{text.Bold}

	writer.SetStyle(style)
	writer.SetColumnConfigs(columnConfigs)
	writer.SuppressTrailingSpaces()

	return writer.Render()
}
