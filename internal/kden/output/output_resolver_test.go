package output_test

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("format output", func() {
	Context("when ResolveFormat is called", func() {
		DescribeTable("should work correctly with valid encoded object",
			func(outputFormat string, encodedObject []byte) {
				cfg.Config.Output = outputFormat
				_, err := output.ResolveFormat(encodedObject, nil)

				Expect(err).NotTo(HaveOccurred())
			},
			Entry("with valid JSON to YAML format", "yaml", []byte("{\"test\": \"test\", \"tested\": \"tested\"}")),
			Entry("with valid YAML to YAML format", "yaml", []byte("test: test")),
			Entry("with valid JSON to JSON format", "json", []byte("{\"test\": \"test\", \"tested\": \"tested\"}")),
			Entry("with valid YAML to JSON format", "json", []byte("test: test")),
		)

		It("should work correctly with valid table data", func() {
			cfg.Config.Output = "pretty"
			result, err := output.ResolveFormat(struct{}{}, &pretty.TableData{
				Columns: []pretty.Column{{Title: "Column", Width: 20}},
				Rows:    []pretty.Row{{"value"}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("Column"))
			Expect(result).To(ContainSubstring("value"))
		})

		DescribeTable("should throw error with invalid encoded object",
			func(outputFormat string, encodedObject []byte, errorMessage string) {
				cfg.Config.Output = outputFormat
				_, err := output.ResolveFormat(encodedObject, nil)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errorMessage))
			},
			Entry("with invalid format configuration", "xml", []byte("test: test"), "error with provided output format: xml"),
			Entry("with plain for a non-version command", "plain", []byte("test: test"),
				"plain output is only supported by the version command"),
			Entry("with invalid object input", "yaml", []byte("{\"test\": \"t\"est\"}"),
				"error occurred during parse of object to map: {\"test\": \"t\"est\"} : yaml: did not find expected ',' or '}'"),
		)

		It("requires table data for pretty output", func() {
			cfg.Config.Output = "pretty"

			_, err := output.ResolveFormat(struct{}{}, nil)

			Expect(err).To(MatchError("pretty output requires table data"))
		})
	})
})
