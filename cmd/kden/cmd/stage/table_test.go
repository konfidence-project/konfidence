package stage

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("listTable", func() {
	It("maps stages with active version details", func() {
		activeVersion := &apiclient.StageVersion{Id: "sv-1", Status: "Ready"}
		result := listTable(&apiclient.StageList{Data: []apiclient.Stage{
			{Id: "stage-a", Name: "Stage A", LandscapeId: "landscape-a", ActiveStageVersion: activeVersion},
			{Id: "stage-b", Name: "Stage B", LandscapeId: "landscape-b"},
		}})

		Expect(result.Columns).To(Equal([]pretty.Column{
			{Title: "ID", Width: len("stage-a")},
			{Title: "Name", Width: len("Stage A")},
			{Title: "Landscape", Width: len("landscape-a")},
			{Title: "Active Version", Width: len("Active Version")},
			{Title: "Status", Width: len("Status")},
		}))
		Expect(result.Rows).To(Equal([]pretty.Row{
			{"stage-a", "Stage A", "landscape-a", "sv-1", "Ready"},
			{"stage-b", "Stage B", "landscape-b", "-", "-"},
		}))
	})
})
