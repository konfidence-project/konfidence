package pretty_test

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func captureStdout(fn func()) string {
	original := os.Stdout

	reader, writer, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())

	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	Expect(writer.Close()).To(Succeed())

	output, err := io.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())

	return string(output)
}

var _ = Describe("FormatTable", func() {
	twoColumns := func(_ interface{}) *pretty.TableData {
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
		var formatErr error

		output := captureStdout(func() {
			formatErr = pretty.FormatTable(twoColumns, nil)
		})

		Expect(formatErr).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("Field"))
		Expect(output).To(ContainSubstring("Value"))
		Expect(output).To(ContainSubstring("Version"))
		Expect(output).To(ContainSubstring("v1.2.3"))
		Expect(output).To(ContainSubstring("a faint hint"))
	})

	It("renders a borderless table", func() {
		output := captureStdout(func() {
			Expect(pretty.FormatTable(twoColumns, nil)).To(Succeed())
		})

		Expect(output).NotTo(ContainSubstring("|"))
		Expect(output).NotTo(ContainSubstring("+"))
		Expect(output).NotTo(ContainSubstring("┌"))
		Expect(output).NotTo(ContainSubstring("│"))
	})

	It("does not emit TUI navigation help", func() {
		output := captureStdout(func() {
			Expect(pretty.FormatTable(twoColumns, nil)).To(Succeed())
		})

		Expect(output).NotTo(ContainSubstring("↑/k"))
	})

	It("does not pad rows to their maximum configured widths", func() {
		output := captureStdout(func() {
			Expect(pretty.FormatTable(twoColumns, nil)).To(Succeed())
		})

		for _, line := range strings.Split(output, "\n") {
			Expect(len(line)).To(BeNumerically("<", 100))
		}
	})

	It("wraps values at the configured maximum column width", func() {
		const (
			columnWidth = 10
			cellPadding = 2
		)

		longValue := func(_ interface{}) *pretty.TableData {
			return &pretty.TableData{
				Columns: []pretty.Column{
					{Title: "Field", Width: columnWidth},
				},
				Rows: []pretty.Row{
					{"a value longer than ten characters"},
				},
			}
		}

		output := captureStdout(func() {
			Expect(pretty.FormatTable(longValue, nil)).To(Succeed())
		})

		for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
			visibleWidth := text.StringWidthWithoutEscSequences(line)
			Expect(visibleWidth).To(BeNumerically("<=", columnWidth+cellPadding))
		}
	})

	It("surfaces model errors", func() {
		badModel := func(_ any) *pretty.TableData {
			return &pretty.TableData{Err: errors.New("boom")}
		}

		err := pretty.FormatTable(badModel, nil)
		Expect(err).To(MatchError(ContainSubstring("boom")))
	})

	It("renders each section under its heading", func() {
		sectioned := func(_ any) *pretty.TableData {
			columns := []pretty.Column{
				{Title: "Promotion ID", Width: 20},
				{Title: "Detail", Width: 20},
			}

			return &pretty.TableData{
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
			}
		}

		output := captureStdout(func() {
			Expect(pretty.FormatTable(sectioned, nil)).To(Succeed())
		})

		Expect(output).To(ContainSubstring("config-a (src → tgt)"))
		Expect(output).To(ContainSubstring("config-b (src → tgt)"))
		Expect(output).To(ContainSubstring("promo-1"))
		Expect(output).To(ContainSubstring("promo-2"))
	})
})
