package landscape

import (
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("listTable", func() {
	It("maps landscapes to ID and name rows", func() {
		result := listTable(&apiclient.LandscapeList{Data: []apiclient.Landscape{
			{Id: "landscape-a", Name: "Landscape A"},
			{Id: "landscape-b", Name: "Landscape B"},
		}})

		Expect(result.Columns).To(Equal([]pretty.Column{
			{Title: "ID", Width: len("landscape-a")},
			{Title: "Name", Width: len("Landscape A")},
		}))
		Expect(result.Rows).To(Equal([]pretty.Row{
			{"landscape-a", "Landscape A"},
			{"landscape-b", "Landscape B"},
		}))
	})
})
