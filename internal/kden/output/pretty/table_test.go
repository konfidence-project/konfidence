package pretty_test

import (
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RenderTable", func() {
	twoColumns := func() *pretty.TableData {
		return &pretty.TableData{
			Columns: []pretty.Column{
				{Title: "Field", Width: 12},
				{Title: "Value", Width: 20},
			},
			Rows: []pretty.Row{
				{"Version", "v1.2.3"},
				{"Commit", "abc1234"},
			},
			Footer: "a faint hint",
		}
	}

	It("renders headers, rows, and the footer", func() {
		output, err := pretty.RenderTable(twoColumns())

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("Field"))
		Expect(output).To(ContainSubstring("Value"))
		Expect(output).To(ContainSubstring("Version"))
		Expect(output).To(ContainSubstring("v1.2.3"))
		Expect(output).To(ContainSubstring("a faint hint"))
	})

	It("renders a borderless table", func() {
		output, err := pretty.RenderTable(twoColumns())

		Expect(err).NotTo(HaveOccurred())
		Expect(output).NotTo(ContainSubstring("|"))
		Expect(output).NotTo(ContainSubstring("+"))
		Expect(output).NotTo(ContainSubstring("┌"))
		Expect(output).NotTo(ContainSubstring("│"))
	})

	It("does not emit TUI navigation help", func() {
		output, err := pretty.RenderTable(twoColumns())

		Expect(err).NotTo(HaveOccurred())
		Expect(output).NotTo(ContainSubstring("↑/k"))
	})

	It("does not pad rows to their maximum configured widths", func() {
		output, err := pretty.RenderTable(twoColumns())

		Expect(err).NotTo(HaveOccurred())
		for _, line := range strings.Split(output, "\n") {
			Expect(len(line)).To(BeNumerically("<", 100))
		}
	})

	It("wraps values at the configured maximum column width", func() {
		const (
			columnWidth = 10
			cellPadding = 2
		)

		output, err := pretty.RenderTable(&pretty.TableData{
			Columns: []pretty.Column{
				{Title: "Field", Width: columnWidth},
			},
			Rows: []pretty.Row{
				{"a value longer than ten characters"},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
			visibleWidth := text.StringWidthWithoutEscSequences(line)
			Expect(visibleWidth).To(BeNumerically("<=", columnWidth+cellPadding))
		}
	})

	It("requires table data", func() {
		_, err := pretty.RenderTable(nil)

		Expect(err).To(MatchError("pretty output requires table data"))
	})

	It("renders each section under its heading", func() {
		columns := []pretty.Column{
			{Title: "Promotion ID", Width: 20},
			{Title: "Detail", Width: 20},
		}

		output, err := pretty.RenderTable(&pretty.TableData{
			Sections: []pretty.TableSection{
				{
					Heading: "config-a (src → tgt)",
					Columns: columns,
					Rows: []pretty.Row{
						{"promo-1", "detail-a"},
					},
				},
				{
					Heading: "config-b (src → tgt)",
					Columns: columns,
					Rows: []pretty.Row{
						{"promo-2", "detail-b"},
					},
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("config-a (src → tgt)"))
		Expect(output).To(ContainSubstring("config-b (src → tgt)"))
		Expect(output).To(ContainSubstring("promo-1"))
		Expect(output).To(ContainSubstring("promo-2"))
	})
})
