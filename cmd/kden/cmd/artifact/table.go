package artifact

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func contentTable(content *apiclient.ArtifactContent) *pretty.TableData {
	builder := pretty.NewTable()

	summary := builder.Section(content.Ref).Columns("Field", "Value")
	summary.Row().String("Field", "Type").String("Value", content.Type)
	summary.Row().String("Field", "Allow Reuse").Value("Value", content.AllowReuse)

	if content.Resources != nil && len(*content.Resources) > 0 {
		res := builder.Section("Resources").Columns("Name", "Type")
		for _, r := range *content.Resources {
			res.Row().
				String("Name", r.Name).
				String("Type", r.Type)
		}
	}

	if content.Tasks != nil && len(*content.Tasks) > 0 {
		tasks := builder.Section("Tasks").Columns("Name", "Type", "Depends On")
		for _, t := range *content.Tasks {
			row := tasks.Row().
				String("Name", t.Name).
				String("Type", t.Type)

			if t.DependsOn != nil {
				row.List("Depends On", *t.DependsOn)
			} else {
				row.String("Depends On", "-")
			}
		}
	}

	return builder.TableData()
}
