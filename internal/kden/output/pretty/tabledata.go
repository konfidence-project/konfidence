package pretty

import (
	"fmt"
	"strings"
	"sync"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/build"
)

var (
	modelFuncMap map[string]ModelFunc
	once         sync.Once
)

func GetModelFuncMap() map[string]ModelFunc {
	once.Do(func() {
		modelFuncMap = map[string]ModelFunc{
			"validate":                     validateModelFunc,
			"artifact-deployment-list":     artifactDeploymentListModelFunc,
			"project-list":                 projectListModelFunc,
			"landscape-list":               landscapeListModelFunc,
			"stage-list":                   stageListModelFunc,
			"vector-deployment-list":       vectorDeploymentListModelFunc,
			"vector-promotion-config":      vectorPromotionConfigModelFunc,
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

func buildTableData[T any](data interface{}, command string, columns []Column, rowMapper func(T) []Row) *TableData {
	v, errData := parseData[T](data, command)
	if errData != nil {
		return errData
	}
	return &TableData{
		Columns: columns,
		Rows:    rowMapper(v),
	}
}

func mapRows[T any](items []T, mapper func(T) Row) []Row {
	rows := make([]Row, 0, len(items))
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
		[]Column{
			{Title: "Field", Width: 12},
			{Title: "Value", Width: 60},
		},
		func(info build.Info) []Row {
			return []Row{
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
		[]Column{
			{Title: "File", Width: 40},
			{Title: "Path", Width: 80},
			{Title: "Message", Width: 80},
		},
		func(msg []output.SchemaValidationError) []Row {
			return mapRows(msg, func(e output.SchemaValidationError) Row {
				return Row{e.File, e.Path, e.Message}
			})
		},
	)
}

func projectListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.ProjectList](data, "project-list",
		[]Column{
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 40},
		},
		func(list *apiclient.ProjectList) []Row {
			return mapRows(list.Data, func(p apiclient.Project) Row {
				return Row{p.Id, p.Name}
			})
		},
	)
}

func landscapeListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.LandscapeList](data, "landscape-list",
		[]Column{
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 40},
		},
		func(list *apiclient.LandscapeList) []Row {
			return mapRows(list.Data, func(l apiclient.Landscape) Row {
				return Row{l.Id, l.Name}
			})
		},
	)
}

func stageListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.StageList](data, "stage-list",
		[]Column{
			{Title: ColumnID, Width: 40},
			{Title: ColumnName, Width: 30},
			{Title: ColumnLandscape, Width: 30},
			{Title: "Active Version", Width: 40},
			{Title: ColumnStatus, Width: 20},
		},
		func(list *apiclient.StageList) []Row {
			return mapRows(list.Data, func(s apiclient.Stage) Row {
				activeVersion, status := "-", "-"
				if s.ActiveStageVersion != nil {
					activeVersion = s.ActiveStageVersion.Id
					status = string(s.ActiveStageVersion.Status)
				}
				return Row{s.Id, s.Name, s.LandscapeId, activeVersion, status}
			})
		},
	)
}

func vectorDeploymentListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.VectorDeploymentList](data, "vector-deployment-list",
		[]Column{
			{Title: ColumnID, Width: 40},
			{Title: "Stage", Width: 30},
			{Title: ColumnLandscape, Width: 30},
			{Title: ColumnStatus, Width: 20},
			{Title: ColumnVector, Width: 40},
		},
		func(list *apiclient.VectorDeploymentList) []Row {
			return mapRows(list.Data, func(d apiclient.VectorDeployment) Row {
				return Row{
					d.Id, d.StageId, d.LandscapeId, string(d.Status),
					fmt.Sprintf("%s:%s", d.Vector.ComponentName, d.Vector.ComponentVersion),
				}
			})
		},
	)
}

func artifactDeploymentListModelFunc(data interface{}) *TableData {
	return buildTableData[*apiclient.ArtifactDeploymentList](data, "artifact-deployment-list",
		[]Column{
			{Title: ColumnID, Width: 40},
			{Title: ColumnLandscape, Width: 30},
			{Title: ColumnStatus, Width: 20},
			{Title: "Artifact", Width: 40},
			{Title: "Stages", Width: 40},
			{Title: "Vector Deployments", Width: 40},
		},
		func(list *apiclient.ArtifactDeploymentList) []Row {
			return mapRows(list.Data, func(d apiclient.ArtifactDeployment) Row {
				return Row{
					d.Id,
					d.LandscapeId,
					string(d.Status),
					formatComponentReference(d.Artifact),
					strings.Join(d.StageIds, ", "),
					strings.Join(d.VectorDeploymentIds, ", "),
				}
			})
		},
	)
}

func vectorPromotionConfigModelFunc(data interface{}) *TableData {
	config, errData := parseData[*apiclient.VectorPromotionConfig](data, "vector-promotion-config")
	if errData != nil {
		return errData
	}

	return vectorPromotionConfigTableData([]apiclient.VectorPromotionConfig{*config})
}

func vectorPromotionConfigListModelFunc(data interface{}) *TableData {
	list, errData := parseData[*apiclient.VectorPromotionConfigList](data, "vector-promotion-config-list")
	if errData != nil {
		return errData
	}

	return vectorPromotionConfigTableData(list.Data)
}

func vectorPromotionConfigTableData(configs []apiclient.VectorPromotionConfig) *TableData {

	columns := []Column{
		{Title: ColumnID, Width: 40},
		{Title: "Source", Width: 24},
		{Title: "Target", Width: 24},
		{Title: ColumnVector, Width: 30},
		{Title: ColumnStatus, Width: 16},
	}

	sections := make([]TableSection, 0, len(configs))
	for _, cfg := range configs {
		rows := make([]Row, 0, len(cfg.Promotions))
		for _, p := range cfg.Promotions {
			status := "-"
			if p.Status != nil {
				status = string(*p.Status)
			}
			rows = append(rows, Row{p.Id, p.Source.Name, p.Target.Name, p.Vector, status})
		}
		sections = append(sections, TableSection{
			Heading: fmt.Sprintf("%s (%s → %s)", cfg.Id, cfg.Source.Name, cfg.Target.Name),
			Columns: columns,
			Rows:    rows,
		})
	}

	return &TableData{Sections: sections}
}

func formatComponentReference(ref apiclient.ComponentReference) string {
	return fmt.Sprintf("%s:%s", ref.ComponentName, ref.ComponentVersion)
}
