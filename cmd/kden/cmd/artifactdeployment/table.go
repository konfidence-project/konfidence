package artifactdeployment

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func listTable(list *apiclient.ArtifactDeploymentList) *pretty.TableData {
	builder := pretty.NewTable()
	for _, d := range list.Data {
		builder.Row().
			String("ID", d.Id).
			String("Landscape", d.LandscapeId).
			String("Status", string(d.Status)).
			Component("Artifact", d.Artifact.ComponentName, d.Artifact.ComponentVersion).
			List("Stages", d.StageIds).
			List("Vector Deployments", d.VectorDeploymentIds)
	}
	return builder.TableData()
}
