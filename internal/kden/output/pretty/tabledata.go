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
			"validate":                     validateModelFunc,
			"project-list":                 projectListModelFunc,
			"landscape-list":               landscapeListModelFunc,
			"stage-list":                   stageListModelFunc,
			"vector-deployment-list":       vectorDeploymentListModelFunc,
			"vector-promotion-config-list": vectorPromotionConfigListModelFunc,
			"version":                      versionModelFunc,
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

const (
	ColumnID        = "ID"
	ColumnName      = "Name"
	ColumnStatus    = "Status"
	ColumnLandscape = "Landscape"
	ColumnVector    = "Vector"
)

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
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 40},
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
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 40},
		},
		func(list *apiclient.LandscapeList) []table.Row {
			return mapRows(list.Data, func(l apiclient.Landscape) table.Row {
				return table.Row{l.Id, l.Name}
			})
		},
	)
}

func stageListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.StageList](data, "stage-list",
		[]table.Column{
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 30},
			{Title: ColumnLandscape, Width: 30},
			{Title: "Active Version", Width: 40},
			{Title: ColumnStatus, Width: 20},
		},
		func(list *apiclient.StageList) []table.Row {
			return mapRows(list.Data, func(s apiclient.Stage) table.Row {
				activeVersion, status := "-", "-"
				if s.ActiveStageVersion != nil {
					activeVersion = s.ActiveStageVersion.Id
					status = string(s.ActiveStageVersion.Status)
				}
				return table.Row{s.Id, s.Name, s.LandscapeId, activeVersion, status}
			})
		},
	)
}

func vectorDeploymentListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.VectorDeploymentList](data, "vector-deployment-list",
		[]table.Column{
			{Title: ColumnID, Width: 40},
			{Title: "Stage", Width: 30},
			{Title: ColumnLandscape, Width: 30},
			{Title: ColumnStatus, Width: 20},
			{Title: ColumnVector, Width: 40},
		},
		func(list *apiclient.VectorDeploymentList) []table.Row {
			return mapRows(list.Data, func(d apiclient.VectorDeployment) table.Row {
				return table.Row{
					d.Id, d.StageId, d.LandscapeId, string(d.Status),
					fmt.Sprintf("%s:%s", d.Vector.ComponentName, d.Vector.ComponentVersion),
				}
			})
		},
	)
}

func vectorPromotionConfigListModelFunc(data interface{}) *TableData {
	list, errData := parseData[*apiclient.VectorPromotionConfigList](data, "vector-promotion-config-list")
	if errData != nil {
		return errData
	}

	columns := []table.Column{
		{Title: ColumnID, Width: 40},
		{Title: "Source", Width: 24},
		{Title: "Target", Width: 24},
		{Title: ColumnVector, Width: 30},
		{Title: ColumnStatus, Width: 16},
	}

	sections := make([]TableSection, 0, len(list.Data))
	for _, cfg := range list.Data {
		rows := make([]table.Row, 0, len(cfg.Promotions))
		for _, p := range cfg.Promotions {
			status := "-"
			if p.Status != nil {
				status = string(*p.Status)
			}
			rows = append(rows, table.Row{p.Id, p.Source.Name, p.Target.Name, p.Vector, status})
		}
		sections = append(sections, TableSection{
			Heading: fmt.Sprintf("%s (%s → %s)", cfg.Id, cfg.Source.Name, cfg.Target.Name),
			Columns: columns,
			Rows:    rows,
		})
	}

	return &TableData{Sections: sections}
}
