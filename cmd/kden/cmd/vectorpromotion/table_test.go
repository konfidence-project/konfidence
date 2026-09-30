package vectorpromotion

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("vector promotion table mappings", func() {
	promoting := apiclient.VectorPromotionStatus("Promoting")
	config := apiclient.VectorPromotionConfig{
		Id:     "cfg-a",
		Source: apiclient.PromotionSourceReference{Name: "src-a"},
		Target: apiclient.PromotionTargetReference{Name: "tgt-a"},
		Promotions: []apiclient.VectorPromotion{
			{
				Id:     "promo-1",
				Source: apiclient.PromotionSourceReference{Name: "psrc-1"},
				Target: apiclient.PromotionTargetReference{Name: "ptgt-1"},
				Vector: "1.0.0",
				Status: &promoting,
			},
			{
				Id:     "promo-2",
				Source: apiclient.PromotionSourceReference{Name: "psrc-2"},
				Target: apiclient.PromotionTargetReference{Name: "ptgt-2"},
				Vector: "2.0.0",
			},
		},
	}

	promotionColumns := []pretty.Column{
		{Title: "ID", Width: len("promo-1")},
		{Title: "Source", Width: len("psrc-1")},
		{Title: "Target", Width: len("ptgt-1")},
		{Title: "Vector", Width: len("Vector")},
		{Title: "Status", Width: len("Promoting")},
	}

	It("maps one config to a single section", func() {
		result := configTable(&config)

		Expect(result.Sections).To(Equal([]pretty.TableSection{
			{
				Heading: "cfg-a (src-a → tgt-a)",
				Columns: promotionColumns,
				Rows: []pretty.Row{
					{"promo-1", "psrc-1", "ptgt-1", "1.0.0", "Promoting"},
					{"promo-2", "psrc-2", "ptgt-2", "2.0.0", "-"},
				},
			},
		}))
	})

	It("maps config lists to one section per config", func() {
		emptyConfig := apiclient.VectorPromotionConfig{
			Id:     "cfg-b",
			Source: apiclient.PromotionSourceReference{Name: "src-b"},
			Target: apiclient.PromotionTargetReference{Name: "tgt-b"},
		}

		result := configListTable(&apiclient.VectorPromotionConfigList{
			Data: []apiclient.VectorPromotionConfig{config, emptyConfig},
		})

		Expect(result.Sections).To(HaveLen(2))
		Expect(result.Sections[0].Heading).To(Equal(fmt.Sprintf("%s (%s → %s)", config.Id, config.Source.Name, config.Target.Name)))
		Expect(result.Sections[0].Rows).To(HaveLen(2))
		Expect(result.Sections[1].Heading).To(Equal("cfg-b (src-b → tgt-b)"))
		Expect(result.Sections[1].Rows).To(BeEmpty())
	})
})
