package stage

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func listTable(list *apiclient.StageList) *pretty.TableData {
	builder := pretty.NewTable().Columns("ID", "Name", "Landscape", "Active Version", "Status")
	for _, s := range list.Data {
		activeVersion, status := "-", "-"
		if s.ActiveStageVersion != nil {
			activeVersion = s.ActiveStageVersion.Id
			status = string(s.ActiveStageVersion.Status)
		}
		builder.Row().
			String("ID", s.Id).
			String("Name", s.Name).
			String("Landscape", s.LandscapeId).
			String("Active Version", activeVersion).
			String("Status", status)
	}
	return builder.TableData()
}
