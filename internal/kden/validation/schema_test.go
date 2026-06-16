package validation_test

import (
	"github.com/konfidence-project/konfidence/internal/kden/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const validYAML = `
components:
  - name: my-component
    version: 1.0.0
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
`

const missingManifestResourceYAML = `
components:
  - name: my-component
    version: 1.0.0
    resources:
      - name: my-artifact
        type: some.other.type
        input:
          type: file/v1
`

const missingInputTypeYAML = `
components:
  - name: my-component
    version: 1.0.0
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input: {}
`

const invalidInputTypeYAML = `
components:
  - name: my-component
    version: 1.0.0
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: unsupported/v1
`

const validJSON = `{
  "components": [{
    "name": "my-component",
    "version": "1.0.0",
    "resources": [{
      "name": "my-artifact",
      "type": "cloud.konfidence.artifact.manifest",
      "input": { "type": "file/v1" }
    }]
  }]
}`

const missingManifestResourceJSON = `{
  "components": [{
     "resources": [{"name": "x", "type": "other", "input": {"type": "file/v1"}}]
  }]
}`

const missingInputTypeJSON = `{
  "components": [{
     "resources": [{"name": "x", "type": "other", "input": {}}]
  }]
}`

const invalidInputTypeJSON = `{
  "components": [{
	 "resources": [{"name": "x", "type": "cloud.konfidence.artifact.manifest", "input": {"type": "bad/v1"}}]
  }]
}`

var _ = Describe("ValidateRaw", func() {

	DescribeTable("valid input",
		func(fn func([]byte) error, input string) {
			Expect(fn([]byte(input))).NotTo(HaveOccurred())
		},
		Entry("YAML: valid input", validation.ValidateRawYAML, validYAML),
		Entry("JSON: valid input", validation.ValidateRawJSON, validJSON),
	)

	DescribeTable("invalid input",
		func(fn func([]byte) error, input string, errContains string) {
			err := fn([]byte(input))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errContains))
		},
		Entry("YAML: malformed document", validation.ValidateRawYAML, ":\tinvalid: yaml: [", "failed to unmarshal YAML"),
		Entry("YAML: missing manifest resource", validation.ValidateRawYAML, missingManifestResourceYAML, ""),
		Entry("YAML: missing input type", validation.ValidateRawYAML, missingInputTypeYAML, ""),
		Entry("YAML: invalid input type", validation.ValidateRawYAML, invalidInputTypeYAML, ""),
		Entry("JSON: malformed document", validation.ValidateRawJSON, `{not valid json`, "unmarshal"),
		Entry("JSON: missing manifest resource", validation.ValidateRawJSON, missingManifestResourceJSON, ""),
		Entry("JSON: missing input type", validation.ValidateRawJSON, missingInputTypeJSON, ""),
		Entry("JSON: invalid input type", validation.ValidateRawJSON, invalidInputTypeJSON, ""),
	)

})
