package vectorpromotion

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
)

func configTable(config *apiclient.VectorPromotionConfig) *pretty.TableData {
	return configsTable([]apiclient.VectorPromotionConfig{*config})
}

func configListTable(list *apiclient.VectorPromotionConfigList) *pretty.TableData {
	return configsTable(list.Data)
}

func configsTable(configs []apiclient.VectorPromotionConfig) *pretty.TableData {
	builder := pretty.NewTable()
	for _, cfg := range configs {
		section := builder.Section(fmt.Sprintf("%s (%s → %s)", cfg.Id, cfg.Source.Name, cfg.Target.Name)).
			Columns("ID", "Source", "Target", "Vector", "Status")
		for _, p := range cfg.Promotions {
			status := "-"
			if p.Status != nil {
				status = string(*p.Status)
			}
			section.Row().
				String("ID", p.Id).
				String("Source", p.Source.Name).
				String("Target", p.Target.Name).
				String("Vector", p.Vector).
				String("Status", status)
		}
	}
	return builder.TableData()
}
