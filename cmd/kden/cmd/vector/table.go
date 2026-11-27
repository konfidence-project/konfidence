package vector

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func contentTable(list *apiclient.VectorContentList) *pretty.TableData {
	builder := pretty.NewTable()
	for _, c := range list.Data {
		comps := builder.Section(c.Ref).Columns("Component", "Version")
		for _, comp := range c.Components {
			comps.Row().
				String("Component", comp.ComponentName).
				String("Version", comp.Version)
		}

		if c.Config == nil {
			continue
		}

		if c.Config.Features != nil && len(*c.Config.Features) > 0 {
			feat := builder.Section("Features").Columns("Feature", "Value")
			for k, v := range *c.Config.Features {
				feat.Row().String("Feature", k).Value("Value", v)
			}
		}

		if c.Config.Authored != nil && len(*c.Config.Authored) > 0 {
			auth := builder.Section("Authored").Columns("Key", "Value")
			for k, v := range *c.Config.Authored {
				auth.Row().String("Key", k).Value("Value", v)
			}
		}
	}
	return builder.TableData()
}

func diffTable(diff *apiclient.VectorDiff) *pretty.TableData {
	builder := pretty.NewTable()

	comps := builder.Section("Components").Columns("Component", "Change", "Left", "Right")
	for _, c := range diff.Components {
		comps.Row().
			String("Component", c.ComponentName).
			String("Change", string(c.ChangeType)).
			String("Left", stringOrDash(c.LeftVersion)).
			String("Right", stringOrDash(c.RightVersion))
	}

	if diff.Config != nil {
		if diff.Config.Features != nil && len(*diff.Config.Features) > 0 {
			feat := builder.Section("Features").Columns("Feature", "Change", "Left", "Right")
			for _, e := range *diff.Config.Features {
				feat.Row().
					String("Feature", e.Key).
					String("Change", string(e.ChangeType)).
					Value("Left", e.LeftValue).
					Value("Right", e.RightValue)
			}
		}

		if diff.Config.Authored != nil && len(*diff.Config.Authored) > 0 {
			auth := builder.Section("Authored").Columns("Authored Key", "Change", "Left", "Right")
			for _, e := range *diff.Config.Authored {
				auth.Row().
					String("Authored Key", e.Key).
					String("Change", string(e.ChangeType)).
					Value("Left", e.LeftValue).
					Value("Right", e.RightValue)
			}
		}
	}

	return builder.TableData()
}

func stringOrDash(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}
