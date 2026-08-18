package ocm_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
)

const constructorYAML = `
components:
  - name: github.com/my-org/my-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-image
        version: v1.0.0
        type: ociImage
        relation: local
        access:
          type: ociArtifact
          imageReference: my-org/my-image:1.0.0
`

const constructorYAMLWithEnv = `
components:
  - name: github.com/my-org/my-component
    version: ${TEST_COMPONENT_VERSION}
    provider:
      name: my-org
    resources:
      - name: my-image
        version: $TEST_COMPONENT_VERSION
        type: ociImage
        relation: local
        access:
          type: ociArtifact
          imageReference: my-org/my-image:1.0.0
`

var _ = Describe("ParseComponentConstructor", func() {
	const filePath = "test-constructor.yaml"

	Context("when a well-formed constructor is provided", func() {
		It("should return a ComponentConstructor without error", func() {
			result, err := ocm.ParseComponentConstructor(constructorYAML, filePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Components).To(HaveLen(1))
			Expect(result.Components[0].Name).To(Equal("github.com/my-org/my-component"))
			Expect(result.Components[0].Version).To(Equal("v1.0.0"))
			Expect(result.Components[0].Provider.Name).To(Equal("my-org"))
		})
	})

	Context("when ${VAR} syntax is used", func() {
		It("should expand the variable from the environment", func() {
			version := "v9.9.9"
			GinkgoT().Setenv("TEST_COMPONENT_VERSION", version)

			result, err := ocm.ParseComponentConstructor(constructorYAMLWithEnv, filePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Components[0].Version).To(Equal(version))
			Expect(result.Components[0].Resources[0].Version).To(Equal(version))

		})
	})

	Context("when a referenced variable is not set", func() {
		It("should expand to an empty string causing a schema validation error", func() {
			result, err := ocm.ParseComponentConstructor(constructorYAMLWithEnv, filePath)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%q", filePath)))
		})
	})

	Context("when the input is invalid YAML", func() {
		It("should return an error containing the file path", func() {
			result, err := ocm.ParseComponentConstructor(`invalid: yaml: content: [`, filePath)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%q", filePath)))
		})
	})

})
