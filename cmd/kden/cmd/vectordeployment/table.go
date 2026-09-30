package vectordeployment

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func listTable(list *apiclient.VectorDeploymentList) *pretty.TableData {
	builder := pretty.NewTable().Columns("ID", "Stage", "Landscape", "Status", "Vector")
	for _, d := range list.Data {
		builder.Row().
			String("ID", d.Id).
			String("Stage", d.StageId).
			String("Landscape", d.LandscapeId).
			String("Status", string(d.Status)).
			Component("Vector", d.Vector.ComponentName, d.Vector.ComponentVersion)
	}
	return builder.TableData()
}
