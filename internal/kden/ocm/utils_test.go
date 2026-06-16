package ocm_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
)

var _ = Describe("ReadConstructorFromFile", func() {

	Context("when the file exists and contains a valid constructor", func() {
		It("returns the parsed ComponentConstructor", func() {
			p := filepath.Join(GinkgoT().TempDir(), "constructor.yaml")
			Expect(os.WriteFile(p, []byte(constructorYAML), 0o600)).To(Succeed())

			result, err := ocm.ReadConstructorFromFile(p)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Components).To(HaveLen(1))
			Expect(result.Components[0].Name).To(Equal("github.com/my-org/my-component"))
		})
	})

	Context("when there is an error reading the file", func() {
		It("returns an error when trying to read the file", func() {
			result, err := ocm.ReadConstructorFromFile("/nonexistent/path/constructor.yaml")

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to read constructor file"))
		})
	})

	Context("when the file contains invalid YAML", func() {
		It("returns an error indicating the constructor could not be unmarshalled", func() {
			p := filepath.Join(GinkgoT().TempDir(), "constructor.yaml")
			Expect(os.WriteFile(p, []byte("invalid: yaml: content: ["), 0o600)).To(Succeed())

			result, err := ocm.ReadConstructorFromFile(p)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshall constructor"))
		})
	})
})
