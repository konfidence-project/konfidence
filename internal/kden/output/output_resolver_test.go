package output_test

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	valout "github.com/konfidence-project/konfidence/internal/kden/validation/output"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("format output", func() {
	Context("when ResolveFormat is called", func() {
		DescribeTable("should work correctly with valid encoded object",
			func(outputFormat string, encodedObject []byte) {
				cfg.Config.Output = outputFormat
				_, err := output.ResolveFormat(encodedObject, "")

				Expect(err).NotTo(HaveOccurred())
			},
			Entry("with valid JSON to YAML format", "yaml", []byte("{\"test\": \"test\", \"tested\": \"tested\"}")),
			Entry("with valid YAML to YAML format", "yaml", []byte("test: test")),
			Entry("with valid JSON to JSON format", "json", []byte("{\"test\": \"test\", \"tested\": \"tested\"}")),
			Entry("with valid YAML to JSON format", "json", []byte("test: test")),
		)

		DescribeTable("should work correctly with valid table",
			func(outputFormat string, command string, data interface{}) {
				cfg.Config.Output = outputFormat
				_, err := output.ResolveFormat(data, command)

				Expect(err).NotTo(HaveOccurred())
			},
			Entry("with valid validation errors and pretty table", "pretty", "validate", []valout.SchemaValidationError{
				{File: "a.yaml", Path: "/spec/name", Message: "required"},
				{File: "b.yaml", Path: "/spec/kind", Message: "invalid enum value"},
			}),
		)

		DescribeTable("should throw error with invalid encoded object",
			func(outputFormat string, encodedObject []byte, errorMessage string) {
				cfg.Config.Output = outputFormat
				_, err := output.ResolveFormat(encodedObject, "")

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errorMessage))
			},
			Entry("with invalid format configuration", "xml", []byte("test: test"), "error with provided output format: xml"),
			Entry("with plain for a non-version command", "plain", []byte("test: test"),
				"plain output is only supported by the version command"),
			Entry("with invalid object input", "yaml", []byte("{\"test\": \"t\"est\"}"),
				"error occurred during parse of object to map: {\"test\": \"t\"est\"} : yaml: did not find expected ',' or '}'"),
		)
	})
})
