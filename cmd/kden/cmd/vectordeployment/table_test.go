package vectordeployment

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("listTable", func() {
	It("maps vector deployments to stage, landscape, status, and vector rows", func() {
		result := listTable(&apiclient.VectorDeploymentList{Data: []apiclient.VectorDeployment{
			{
				Id: "vd-a", StageId: "stage-a", LandscapeId: "landscape-a", Status: "Ready",
				Vector: apiclient.VectorReference{ComponentName: "comp", ComponentVersion: "1.2.3"},
			},
		}})

		Expect(result.Columns).To(Equal([]pretty.Column{
			{Title: "ID", Width: len("vd-a")},
			{Title: "Stage", Width: len("stage-a")},
			{Title: "Landscape", Width: len("landscape-a")},
			{Title: "Status", Width: len("Status")},
			{Title: "Vector", Width: len("comp:1.2.3")},
		}))
		Expect(result.Rows).To(Equal([]pretty.Row{{"vd-a", "stage-a", "landscape-a", "Ready", "comp:1.2.3"}}))
	})
})
