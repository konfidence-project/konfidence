package project

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func listTable(list *apiclient.ProjectList) *pretty.TableData {
	builder := pretty.NewTable().Columns("ID", "Name")
	for _, p := range list.Data {
		builder.Row().
			String("ID", p.Id).
			String("Name", p.Name)
	}
	return builder.TableData()
}
