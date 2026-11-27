package artifact

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testArtifactRef  = "registry.example.com//test/artifact:1.0.0"
	testArtifactType = "helm-chart"
)

var _ = Describe("artifact content table mappings", func() {
	It("maps the summary, resources and tasks sections", func() {
		dependsOn := []string{"db-migrate", "seed-data"}
		content := apiclient.ArtifactContent{
			Ref:        testArtifactRef,
			Type:       testArtifactType,
			AllowReuse: true,
			Resources: &[]apiclient.ArtifactResource{
				{Name: "artifact-image", Type: "ociImage"},
				{Name: "artifact-chart", Type: "helmChart"},
			},
			Tasks: &[]apiclient.ArtifactTask{
				{Name: "db-migrate", Type: "job"},
				{Name: "deploy", Type: "helm", DependsOn: &dependsOn},
			},
		}

		result := contentTable(&content)

		Expect(result.Sections).To(HaveLen(3))

		summary := result.Sections[0]
		Expect(summary.Heading).To(Equal(content.Ref))
		Expect(summary.Rows).To(Equal([]pretty.Row{
			{"Type", content.Type},
			{"Allow Reuse", "true"},
		}))

		resources := result.Sections[1]
		Expect(resources.Rows).To(Equal([]pretty.Row{
			{(*content.Resources)[0].Name, (*content.Resources)[0].Type},
			{(*content.Resources)[1].Name, (*content.Resources)[1].Type},
		}))

		tasks := result.Sections[2]
		Expect(tasks.Rows).To(Equal([]pretty.Row{
			{(*content.Tasks)[0].Name, (*content.Tasks)[0].Type, "-"},
			{(*content.Tasks)[1].Name, (*content.Tasks)[1].Type, "db-migrate, seed-data"},
		}))
	})

	It("renders allow reuse false in the summary", func() {
		content := apiclient.ArtifactContent{
			Ref:        testArtifactRef,
			Type:       testArtifactType,
			AllowReuse: false,
		}

		result := contentTable(&content)

		Expect(result.Sections[0].Rows).To(Equal([]pretty.Row{
			{"Type", content.Type},
			{"Allow Reuse", "false"},
		}))
	})

	It("omits the resources and tasks sections when absent", func() {
		content := apiclient.ArtifactContent{
			Ref:        testArtifactRef,
			Type:       testArtifactType,
			AllowReuse: true,
		}

		result := contentTable(&content)

		Expect(result.Sections).To(HaveLen(1))
		Expect(result.Sections[0].Heading).To(Equal(content.Ref))
	})

	It("omits the resources and tasks sections when empty", func() {
		content := apiclient.ArtifactContent{
			Ref:        testArtifactRef,
			Type:       testArtifactType,
			AllowReuse: true,
			Resources:  &[]apiclient.ArtifactResource{},
			Tasks:      &[]apiclient.ArtifactTask{},
		}

		result := contentTable(&content)

		Expect(result.Sections).To(HaveLen(1))
	})

	It("renders a dash for a task with no dependencies", func() {
		content := apiclient.ArtifactContent{
			Ref:        testArtifactRef,
			Type:       testArtifactType,
			AllowReuse: true,
			Tasks: &[]apiclient.ArtifactTask{
				{Name: "db-migrate", Type: "job"},
			},
		}

		result := contentTable(&content)

		Expect(result.Sections).To(HaveLen(2))
		Expect(result.Sections[1].Rows).To(Equal([]pretty.Row{
			{"db-migrate", "job", "-"},
		}))
	})
})
