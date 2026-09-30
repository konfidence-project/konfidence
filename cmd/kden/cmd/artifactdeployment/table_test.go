package artifactdeployment

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("listTable", func() {
	It("maps artifact deployments to artifact, stage, and vector deployment rows", func() {
		result := listTable(&apiclient.ArtifactDeploymentList{Data: []apiclient.ArtifactDeployment{
			{
				Id:                  "ad-a",
				LandscapeId:         "landscape-a",
				Status:              "Ready",
				StageIds:            []apiclient.StageId{"stage-a", "stage-b"},
				VectorDeploymentIds: []apiclient.VectorDeploymentId{"vd-a", "vd-b"},
				Artifact: apiclient.ArtifactReference{
					ComponentName:    "artifact",
					ComponentVersion: "1.2.3",
					Repository:       "repo.example.com/artifacts",
				},
			},
		}})

		Expect(result.Columns).To(Equal([]pretty.Column{
			{Title: "ID", Width: len("ad-a")},
			{Title: "Landscape", Width: len("landscape-a")},
			{Title: "Status", Width: len("Status")},
			{Title: "Artifact", Width: len("artifact:1.2.3")},
			{Title: "Stages", Width: len("stage-a, stage-b")},
			{Title: "Vector Deployments", Width: len("Vector Deployments")},
		}))
		Expect(result.Rows).To(Equal([]pretty.Row{{
			"ad-a", "landscape-a", "Ready", "artifact:1.2.3", "stage-a, stage-b", "vd-a, vd-b",
		}}))
	})
})
