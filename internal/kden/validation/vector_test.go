package validation_test

import (
	"github.com/konfidence-project/konfidence/internal/kden/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const vectorFilePath = "vector.yaml"

const validVectorYAML = `
components:
  - name: example.org/vectors/application
    version: 1.0.0
    provider:
      name: example
    componentReferences:
      - name: service
        componentName: example.org/artifacts/service
        version: 1.0.0
      - name: database
        componentName: example.org/artifacts/database
        version: 1.0.0
`

const duplicateArtifactVersionsVectorYAML = `
components:
  - name: example.org/vectors/application
    version: 1.0.0
    provider:
      name: example
    componentReferences:
      - name: service-v1
        componentName: example.org/artifacts/service
        version: 1.0.0
      - name: service-v2
        componentName: example.org/artifacts/service
        version: 2.0.0
`

const exactDuplicateArtifactVectorYAML = `
components:
  - name: example.org/vectors/application
    version: 1.0.0
    provider:
      name: example
    componentReferences:
      - name: service-primary
        componentName: example.org/artifacts/service
        version: 1.0.0
      - name: service-secondary
        componentName: example.org/artifacts/service
        version: 1.0.0
`

const independentVectorsYAML = `
components:
  - name: example.org/vectors/first
    version: 1.0.0
    provider:
      name: example
    componentReferences:
      - name: service
        componentName: example.org/artifacts/service
        version: 1.0.0
  - name: example.org/vectors/second
    version: 1.0.0
    provider:
      name: example
    componentReferences:
      - name: service
        componentName: example.org/artifacts/service
        version: 2.0.0
`

var _ = Describe("ValidateVector", func() {
	It("accepts artifacts with distinct component names", func() {
		errs, err := validation.ValidateVector([]byte(validVectorYAML), vectorFilePath, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(errs).To(BeEmpty())
	})

	It("rejects different versions of the same artifact", func() {
		errs, err := validation.ValidateVector([]byte(duplicateArtifactVersionsVectorYAML), vectorFilePath, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].File).To(Equal(vectorFilePath))
		Expect(errs[0].Path).To(Equal("/components/0/componentReferences/1/componentName"))
		Expect(errs[0].Message).To(ContainSubstring(`artifact "example.org/artifacts/service" must not appear more than once`))
		Expect(errs[0].Message).To(ContainSubstring(`"1.0.0"`))
		Expect(errs[0].Message).To(ContainSubstring(`"2.0.0"`))
	})

	It("rejects the same artifact version referenced more than once", func() {
		errs, err := validation.ValidateVector([]byte(exactDuplicateArtifactVectorYAML), vectorFilePath, nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Message).To(ContainSubstring(`artifact "example.org/artifacts/service" must not appear more than once`))
	})

	It("scopes uniqueness to each top-level vector", func() {
		errs, err := validation.ValidateVector([]byte(independentVectorsYAML), vectorFilePath, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(errs).To(BeEmpty())
	})

	It("preserves constructor parsing errors", func() {
		errs, err := validation.ValidateVector([]byte(":\tinvalid: yaml: ["), vectorFilePath, nil)
		Expect(err).To(HaveOccurred())
		Expect(errs).To(BeNil())
	})
})
