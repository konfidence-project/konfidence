package vector

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testVectorContentRef = "registry.example.com//test/vector:1.0.0"

var _ = Describe("vector content table mappings", func() {
	It("maps one content ref to a components section", func() {
		content := apiclient.VectorContent{
			Ref: testVectorContentRef,
			Components: []apiclient.VectorComponent{
				{ComponentName: "comp-a", Version: "1.0.0"},
				{ComponentName: "comp-b", Version: "2.0.0"},
			},
		}

		result := contentTable(&apiclient.VectorContentList{Data: []apiclient.VectorContent{content}})

		Expect(result.Sections).To(HaveLen(1))
		Expect(result.Sections[0].Heading).To(Equal(content.Ref))
		Expect(result.Sections[0].Rows).To(Equal([]pretty.Row{
			{content.Components[0].ComponentName, content.Components[0].Version},
			{content.Components[1].ComponentName, content.Components[1].Version},
		}))
	})

	It("renders a ref with no components as an empty section", func() {
		content := apiclient.VectorContent{Ref: testVectorContentRef}

		result := contentTable(&apiclient.VectorContentList{Data: []apiclient.VectorContent{content}})

		Expect(result.Sections).To(HaveLen(1))
		Expect(result.Sections[0].Heading).To(Equal(content.Ref))
		Expect(result.Sections[0].Rows).To(BeEmpty())
	})

	It("maps each ref to its own section", func() {
		first := apiclient.VectorContent{
			Ref:        "registry.example.com//test/vector-a:1.0.0",
			Components: []apiclient.VectorComponent{{ComponentName: "a", Version: "1.0.0"}},
		}
		second := apiclient.VectorContent{
			Ref:        "registry.example.com//test/vector-b:2.0.0",
			Components: []apiclient.VectorComponent{{ComponentName: "b", Version: "2.0.0"}},
		}

		result := contentTable(&apiclient.VectorContentList{Data: []apiclient.VectorContent{first, second}})

		Expect(result.Sections).To(HaveLen(2))
		Expect(result.Sections[0].Heading).To(Equal(first.Ref))
		Expect(result.Sections[1].Heading).To(Equal(second.Ref))
	})

	It("adds Features and Authored sections when config is present", func() {
		features := map[string]interface{}{"dark-mode": true}
		authored := map[string]interface{}{"replicaCount": float64(3)}
		content := apiclient.VectorContent{
			Ref:        testVectorContentRef,
			Components: []apiclient.VectorComponent{{ComponentName: "comp-a", Version: "1.0.0"}},
			Config: &apiclient.VectorConfig{
				Features: &features,
				Authored: &authored,
			},
		}

		result := contentTable(&apiclient.VectorContentList{Data: []apiclient.VectorContent{content}})

		Expect(result.Sections).To(HaveLen(3))
		Expect(result.Sections[1].Rows).To(Equal([]pretty.Row{
			{"dark-mode", fmt.Sprint(features["dark-mode"])},
		}))
		Expect(result.Sections[2].Rows).To(Equal([]pretty.Row{
			{"replicaCount", fmt.Sprint(authored["replicaCount"])},
		}))
	})

	It("omits empty config sections", func() {
		features := map[string]interface{}{}
		content := apiclient.VectorContent{
			Ref:    testVectorContentRef,
			Config: &apiclient.VectorConfig{Features: &features},
		}

		result := contentTable(&apiclient.VectorContentList{Data: []apiclient.VectorContent{content}})

		Expect(result.Sections).To(HaveLen(1))
	})
})

var _ = Describe("vector diff table mappings", func() {
	leftVersion := "1.0.0"
	rightVersion := "2.0.0"

	It("maps component diffs with left and right versions", func() {
		component := apiclient.VectorComponentDiff{
			ComponentName: "comp-a",
			ChangeType:    apiclient.Changed,
			LeftVersion:   &leftVersion,
			RightVersion:  &rightVersion,
		}

		result := diffTable(&apiclient.VectorDiff{Components: []apiclient.VectorComponentDiff{component}})

		Expect(result.Sections).To(HaveLen(1))
		Expect(result.Sections[0].Rows).To(Equal([]pretty.Row{
			{component.ComponentName, string(component.ChangeType), *component.LeftVersion, *component.RightVersion},
		}))
	})

	It("renders a dash for the absent side of a removed component", func() {
		component := apiclient.VectorComponentDiff{
			ComponentName: "comp-b",
			ChangeType:    apiclient.Removed,
			LeftVersion:   &leftVersion,
		}

		result := diffTable(&apiclient.VectorDiff{Components: []apiclient.VectorComponentDiff{component}})

		Expect(result.Sections[0].Rows).To(Equal([]pretty.Row{
			{component.ComponentName, string(component.ChangeType), *component.LeftVersion, "-"},
		}))
	})

	It("renders a dash for the absent side of an added component", func() {
		component := apiclient.VectorComponentDiff{
			ComponentName: "comp-c",
			ChangeType:    apiclient.Added,
			RightVersion:  &rightVersion,
		}

		result := diffTable(&apiclient.VectorDiff{Components: []apiclient.VectorComponentDiff{component}})

		Expect(result.Sections[0].Rows).To(Equal([]pretty.Row{
			{component.ComponentName, string(component.ChangeType), "-", *component.RightVersion},
		}))
	})

	It("adds Features and Authored sections for config diffs", func() {
		feature := apiclient.ConfigEntryDiff{Key: "dark-mode", ChangeType: apiclient.Changed, LeftValue: false, RightValue: true}
		authored := apiclient.ConfigEntryDiff{Key: "replica-count", ChangeType: apiclient.Changed, LeftValue: float64(3), RightValue: float64(5)}

		result := diffTable(&apiclient.VectorDiff{
			Components: []apiclient.VectorComponentDiff{},
			Config: &apiclient.VectorConfigDiff{
				Features: &[]apiclient.ConfigEntryDiff{feature},
				Authored: &[]apiclient.ConfigEntryDiff{authored},
			},
		})

		Expect(result.Sections).To(HaveLen(3))
		Expect(result.Sections[1].Rows).To(Equal([]pretty.Row{
			{feature.Key, string(feature.ChangeType), fmt.Sprint(feature.LeftValue), fmt.Sprint(feature.RightValue)},
		}))
		Expect(result.Sections[2].Rows).To(Equal([]pretty.Row{
			{authored.Key, string(authored.ChangeType), fmt.Sprint(authored.LeftValue), fmt.Sprint(authored.RightValue)},
		}))
	})

	It("renders a dash for an absent config value", func() {
		feature := apiclient.ConfigEntryDiff{Key: "new-flag", ChangeType: apiclient.Added, RightValue: true}

		result := diffTable(&apiclient.VectorDiff{
			Components: []apiclient.VectorComponentDiff{},
			Config:     &apiclient.VectorConfigDiff{Features: &[]apiclient.ConfigEntryDiff{feature}},
		})

		Expect(result.Sections[1].Rows).To(Equal([]pretty.Row{
			{feature.Key, string(feature.ChangeType), "-", fmt.Sprint(feature.RightValue)},
		}))
	})

	It("omits config sections when config is nil", func() {
		component := apiclient.VectorComponentDiff{
			ComponentName: "comp-a",
			ChangeType:    apiclient.Unchanged,
			LeftVersion:   &leftVersion,
			RightVersion:  &leftVersion,
		}

		result := diffTable(&apiclient.VectorDiff{Components: []apiclient.VectorComponentDiff{component}})

		Expect(result.Sections).To(HaveLen(1))
	})
})
