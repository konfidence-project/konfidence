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
			"validate":     validateModelFunc,
			"project-list": projectListModelFunc,
			"version":      versionModelFunc,
		}
	})
	return modelFuncMap
}

// updateHint is the footer under the version table. It mirrors the install
// one-liner in cmd/kden/cmd/version; keep the two in sync if the URL changes.
const updateHint = "To update, re-run: curl -fsSL https://konfidence.cloud/install.sh | sh"

func versionModelFunc(data interface{}) *TableData {
	info, ok := data.(build.Info)
	if !ok {
		return &TableData{
			Err: fmt.Errorf("error while creating table for command version: "+
				"expected build.Info, got %T", data),
		}
	}

	return &TableData{
		Columns: []table.Column{
			{Title: "Field", Width: 12},
			{Title: "Value", Width: 60},
		},
		Rows: []table.Row{
			{"Version", info.Version},
			{"Commit", info.Commit},
			{"Go", info.GoVersion},
			{"Platform", info.Platform},
			{"built", info.Date},
		},
		Footer: updateHint,
	}
}

func validateModelFunc(data interface{}) *TableData {
	msg, ok := data.([]output.SchemaValidationError)
	if !ok {
		return &TableData{
			Err: fmt.Errorf("error while creating table for command validate: "+
				"expected []SchemaValidationError, got %T", data),
		}
	}

	columns := []table.Column{
		{Title: "File", Width: 40},
		{Title: "Path", Width: 80},
		{Title: "Message", Width: 80},
	}

	rows := make([]table.Row, 0, len(msg))
	for _, e := range msg {
		rows = append(rows, table.Row{
			e.File,
			e.Path,
			e.Message,
		})
	}

	return &TableData{
		Columns: columns,
		Rows:    rows,
	}
}

func projectListModelFunc(data interface{}) *TableData {
	projects, ok := data.(*apiclient.ProjectList)
	if !ok {
		return &TableData{
			Err: fmt.Errorf("error while creating table for command project-list: "+
				"expected *ProjectList, got %T", data),
		}
	}

	rows := make([]table.Row, 0, len(projects.Data))
	for _, project := range projects.Data {
		rows = append(rows, table.Row{project.Id, project.Name})
	}
	return &TableData{
		Columns: []table.Column{
			{Title: "ID", Width: 40},
			{Title: "Name", Width: 40},
		},
		Rows: rows,
	}
}
