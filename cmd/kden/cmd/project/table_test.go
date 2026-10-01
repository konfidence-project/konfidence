package project

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("listTable", func() {
	It("maps projects to ID and name rows", func() {
		result := listTable(&apiclient.ProjectList{Data: []apiclient.Project{
			{Id: "project-a", Name: "Project A"},
			{Id: "project-b", Name: "Project B"},
		}})

		Expect(result.Columns).To(Equal([]pretty.Column{
			{Title: "ID", Width: len("project-a")},
			{Title: "Name", Width: len("Project A")},
		}))
		Expect(result.Rows).To(Equal([]pretty.Row{
			{"project-a", "Project A"},
			{"project-b", "Project B"},
		}))
	})
})
