package validation_test

import (
	"os"
	"path/filepath"

	"github.com/konfidence-project/konfidence/internal/kden/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const filePath = "test-artifact.yaml"

const validArtifactYAML = `
components:
  - name: github.com/my-org/my-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
`

const invalidTypeArtifactYAML = `
components:
  - name: github.com/my-org/my-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: some.other.type
        input:
          type: file/v1
`

const invalidPathArtifactYAML = `
components:
  - name: github.com/my-org/my-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
      - name: other-resource
        type: some.other.type
        input:
          type: file/v1
`

func artifactYAML(withPath string) string {
	return `
components:
  - name: github.com/my-org/my-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
          path: ` + withPath + `
`
}

func artifactsYAML(withPaths ...string) string {
	return `
components:
  - name: github.com/my-org/first-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
          path: ` + withPaths[0] + `
  - name: github.com/my-org/second-component
    version: v1.0.0
    provider:
      name: my-org
    resources:
      - name: my-artifact
        type: cloud.konfidence.artifact.manifest
        input:
          type: file/v1
          path: ` + withPaths[1] + `
`
}

func writeTempFile(t GinkgoTInterface, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	Expect(os.WriteFile(p, []byte(content), 0o600)).To(Succeed())
	return p
}

var _ = Describe("ValidateArtifact", func() {
	resourceJsonPaths := map[string]bool{}

	Context("with a valid artifact that has no 'path' key", func() {
		It("returns an error for missing 'path' key", func() {
			errs, err := validation.ValidateArtifact([]byte(validArtifactYAML), filePath, resourceJsonPaths)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path"))
			Expect(errs).To(BeNil())
		})
	})

	Context("with invalid YAML that cannot be parsed", func() {
		It("returns an error", func() {
			errs, err := validation.ValidateArtifact([]byte(":\tinvalid: yaml: ["), filePath, resourceJsonPaths)
			Expect(err).To(HaveOccurred())
			Expect(errs).To(BeNil())
		})
	})

	Context("when the YAML fails schema validation", func() {
		It("returns schema validation errors for missing manifest resource type", func() {
			errs, err := validation.ValidateArtifact([]byte(invalidTypeArtifactYAML), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).NotTo(BeEmpty())
		})
	})

	Context("when the manifest resource has a 'path' key", func() {
		It("returns an error when the path file does not exist", func() {
			yaml := artifactYAML("/nonexistent/path/file.json")
			errs, err := validation.ValidateArtifact([]byte(yaml), filePath, resourceJsonPaths)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read file"))
			Expect(errs).To(BeNil())
		})

		It("returns a schema validation error when the referenced file has no 'type' field", func() {
			t := GinkgoT()
			referencedFile := writeTempFile(t, "no-type.json", `{"name": "my-artifact"}`)
			yaml := artifactYAML(referencedFile)

			errs, err := validation.ValidateArtifact([]byte(yaml), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Path).To(ContainSubstring("type"))
			Expect(errs[0].Message).To(ContainSubstring("type"))
		})

		It("returns a schema validation error when the 'type' field in the referenced file is empty", func() {
			t := GinkgoT()
			referencedFile := writeTempFile(t, "empty-type.json", `{"type": ""}`)
			yaml := artifactYAML(referencedFile)

			errs, err := validation.ValidateArtifact([]byte(yaml), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Path).To(ContainSubstring("/type"))
		})

		It("returns no errors when the referenced file has a valid 'type' field", func() {
			t := GinkgoT()
			referencedFile := writeTempFile(t, "valid.json", `{"type": "docker/v1", "name": "my-artifact"}`)
			yaml := artifactYAML(referencedFile)

			errs, err := validation.ValidateArtifact([]byte(yaml), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).To(BeNil())
		})
	})

	Context("when the resource type is not the manifest type", func() {
		It("skips path/type validation for non-manifest resources", func() {
			errs, err := validation.ValidateArtifact([]byte(invalidPathArtifactYAML), filePath, resourceJsonPaths)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path"))
			Expect(errs).To(BeNil())
		})
	})

	Context("when there are multiple components", func() {
		It("validates resources across all components", func() {
			t := GinkgoT()
			validFile := writeTempFile(t, "valid.json", `{"type": "docker/v1"}`)
			referencedFile := writeTempFile(t, "no-type.json", `{"name": "artifact"}`)
			errs, err := validation.ValidateArtifact([]byte(artifactsYAML(validFile, referencedFile)), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).NotTo(BeEmpty())
			Expect(errs[0].Path).To(ContainSubstring("type"))
		})

		It("validates resources across all components with the same path", func() {
			t := GinkgoT()
			validFile := writeTempFile(t, "valid.json", `{"type": "docker/v1"}`)
			errs, err := validation.ValidateArtifact([]byte(artifactsYAML(validFile, validFile)), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).To(BeNil())
		})

		It("validates resources across all components with the different paths", func() {
			t := GinkgoT()
			validFile := writeTempFile(t, "valid.json", `{"type": "docker/v1"}`)
			referencedFile := writeTempFile(t, "valid2.json", `{"type": "docker/v1"}`)
			errs, err := validation.ValidateArtifact([]byte(artifactsYAML(validFile, referencedFile)), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).To(BeNil())
		})
	})

	Context("with empty constructor data", func() {
		It("returns schema validation errors", func() {
			errs, err := validation.ValidateArtifact([]byte(""), filePath, resourceJsonPaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(errs).NotTo(BeEmpty())
		})
	})

})
