package landscape

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func listTable(list *apiclient.LandscapeList) *pretty.TableData {
	builder := pretty.NewTable().Columns("ID", "Name")
	for _, l := range list.Data {
		builder.Row().
			String("ID", l.Id).
			String("Name", l.Name)
	}
	return builder.TableData()
}
