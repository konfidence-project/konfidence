package pretty

import (
	"fmt"
	"sync"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/build"

	"charm.land/bubbles/v2/table"
)

var (
	modelFuncMap map[string]ModelFunc
	once         sync.Once
)

func GetModelFuncMap() map[string]ModelFunc {
	once.Do(func() {
		modelFuncMap = map[string]ModelFunc{
			"validate":       validateModelFunc,
			"project-list":   projectListModelFunc,
			"landscape-list": landscapeListModelFunc,
			"version":        versionModelFunc,
		}
	})
	return modelFuncMap
}

func parseData[T any](data interface{}, command string) (T, *TableData) {
	v, ok := data.(T)
	if !ok {
		return v, &TableData{
			Err: fmt.Errorf("error while creating table for command %s: expected %T, got %T", command, v, data),
		}
	}
	return v, nil
}

func buildTableData[T any](data interface{}, command string, columns []table.Column, rowMapper func(T) []table.Row) *TableData {
	v, errData := parseData[T](data, command)
	if errData != nil {
		return errData
	}
	return &TableData{
		Columns: columns,
		Rows:    rowMapper(v),
	}
}

func mapRows[T any](items []T, mapper func(T) table.Row) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapper(item))
	}
	return rows
}

// updateHint is the footer under the version table. It mirrors the install
// one-liner in cmd/kden/cmd/version; keep the two in sync if the URL changes.
const updateHint = "To update, re-run: curl -fsSL https://konfidence.cloud/install.sh | sh"

func versionModelFunc(data interface{}) *TableData {
	tableData := buildTableData[build.Info](data, "version",
		[]table.Column{
			{Title: "Field", Width: 12},
			{Title: "Value", Width: 60},
		},
		func(info build.Info) []table.Row {
			return []table.Row{
				{"Version", info.Version},
				{"Commit", info.Commit},
				{"Go", info.GoVersion},
				{"Platform", info.Platform},
				{"built", info.Date},
			}
		},
	)
	if tableData.Err != nil {
		return tableData
	}

	tableData.Footer = updateHint
	return tableData
}

func validateModelFunc(data interface{}) *TableData {
	return buildTableData[[]output.SchemaValidationError](data, "validate",
		[]table.Column{
			{Title: "File", Width: 40},
			{Title: "Path", Width: 80},
			{Title: "Message", Width: 80},
		},
		func(msg []output.SchemaValidationError) []table.Row {
			return mapRows(msg, func(e output.SchemaValidationError) table.Row {
				return table.Row{e.File, e.Path, e.Message}
			})
		},
	)
}

func projectListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.ProjectList](data, "project-list",
		[]table.Column{
			{Title: "ID", Width: 40},
			{Title: "Name", Width: 40},
		},
		func(list *apiclient.ProjectList) []table.Row {
			return mapRows(list.Data, func(p apiclient.Project) table.Row {
				return table.Row{p.Id, p.Name}
			})
		},
	)
}

func landscapeListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.LandscapeList](data, "landscape-list",
		[]table.Column{
			{Title: "ID", Width: 40},
			{Title: "Name", Width: 40},
		},
		func(list *apiclient.LandscapeList) []table.Row {
			return mapRows(list.Data, func(l apiclient.Landscape) table.Row {
				return table.Row{l.Id, l.Name}
			})
		},
	)
}
