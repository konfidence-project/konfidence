package pretty

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
)

const defaultColumnWidthMax = 40

// TableBuilder is a small DSL for command packages that need to describe a
// pretty table without manually managing column widths, row ordering, or common
// value-to-string conversions. Command packages still decide which fields are
// shown; the builder only turns those choices into generic TableData.
type TableBuilder struct {
	heading  string
	columns  []string
	rows     []map[string]string
	sections []*TableBuilder
	footer   string
}

// RowBuilder appends values to the row returned by TableBuilder.Row. Each value
// also registers its column name, so callers can either declare columns up front
// with Columns or let the first row define the order incrementally.
type RowBuilder struct {
	table *TableBuilder
	row   map[string]string
}

// NewTable starts a flat table. Use Section for multi-table output such as a
// parent resource with nested child rows.
func NewTable() *TableBuilder {
	return &TableBuilder{}
}

// Columns declares column order before rows are added. Rows can still add new
// columns later, but explicit declaration keeps table shape stable for empty
// result sets.
func (b *TableBuilder) Columns(columns ...string) *TableBuilder {
	for _, column := range columns {
		b.ensureColumn(column)
	}
	return b
}

// Footer adds a note rendered below the table or the final section.
func (b *TableBuilder) Footer(footer string) *TableBuilder {
	b.footer = footer
	return b
}

// Section appends a headed child table. When a table has sections, its own flat
// rows are ignored and each section computes its columns and widths separately.
func (b *TableBuilder) Section(heading string) *TableBuilder {
	section := &TableBuilder{heading: heading}
	b.sections = append(b.sections, section)
	return section
}

// Row starts a row. Chain String, Value, List, or Component calls to fill cells.
func (b *TableBuilder) Row() *RowBuilder {
	row := map[string]string{}
	b.rows = append(b.rows, row)
	return &RowBuilder{table: b, row: row}
}

// TableData returns the render model consumed by RenderTable.
func (b *TableBuilder) TableData() *TableData {
	if len(b.sections) > 0 {
		sections := make([]TableSection, 0, len(b.sections))
		for _, section := range b.sections {
			data := section.tableDataWithoutSections()
			sections = append(sections, TableSection{
				Heading: section.heading,
				Columns: data.Columns,
				Rows:    data.Rows,
			})
		}
		return &TableData{Sections: sections, Footer: b.footer}
	}

	data := b.tableDataWithoutSections()
	data.Footer = b.footer
	return data
}

func (b *TableBuilder) tableDataWithoutSections() *TableData {
	columns := make([]Column, 0, len(b.columns))
	for _, name := range b.columns {
		columns = append(columns, Column{Title: name, Width: b.columnWidth(name)})
	}

	rows := make([]Row, 0, len(b.rows))
	for _, source := range b.rows {
		row := make(Row, 0, len(b.columns))
		for _, column := range b.columns {
			row = append(row, source[column])
		}
		rows = append(rows, row)
	}

	return &TableData{Columns: columns, Rows: rows}
}

func (b *TableBuilder) columnWidth(column string) int {
	width := text.StringWidthWithoutEscSequences(column)
	for _, row := range b.rows {
		if valueWidth := text.StringWidthWithoutEscSequences(row[column]); valueWidth > width {
			width = valueWidth
		}
	}
	if width > defaultColumnWidthMax {
		return defaultColumnWidthMax
	}
	return width
}

// String writes a cell verbatim.
func (r *RowBuilder) String(column, value string) *RowBuilder {
	r.table.ensureColumn(column)
	r.row[column] = value
	return r
}

// Value writes fmt.Sprint(value), using "-" for nil pointers/interfaces.
func (r *RowBuilder) Value(column string, value any) *RowBuilder {
	if value == nil {
		return r.String(column, "-")
	}
	return r.String(column, fmt.Sprint(value))
}

// List writes any slice/array as a comma-separated cell, using "-" for empty
// lists. Non-list values are accepted and rendered as a single item.
func (r *RowBuilder) List(column string, values any) *RowBuilder {
	items := stringifyList(values)
	if len(items) == 0 {
		return r.String(column, "-")
	}
	return r.String(column, strings.Join(items, ", "))
}

// Component writes the common Konfidence component reference shape as
// "name:version" without coupling this generic package to apiclient DTOs.
func (r *RowBuilder) Component(column, name, version string) *RowBuilder {
	return r.String(column, fmt.Sprintf("%s:%s", name, version))
}

func (b *TableBuilder) ensureColumn(column string) {
	if !slices.Contains(b.columns, column) {
		b.columns = append(b.columns, column)
	}
}

func stringifyList(values any) []string {
	if values == nil {
		return nil
	}

	v := reflect.ValueOf(values)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return []string{fmt.Sprint(values)}
	}

	items := make([]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		items = append(items, fmt.Sprint(v.Index(i).Interface()))
	}
	return items
}
