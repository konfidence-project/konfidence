package ocm_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
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

var _ = Describe("GetCredentialGraph", func() {
	// Regression: when a config carries no credentials section,
	// LookupCredentialConfig returns a nil credential config, which ToGraph
	// nil-derefs on. GetCredentialGraph must tolerate that and yield an
	// (anonymous) resolver instead of panicking.
	//
	// GetOcmConfiguration always materializes a non-nil &Config{} for the
	// no-.ocmconfig case, so the reachable input here is an empty (or
	// credentials-free) generic config — not a nil one.
	Context("when the config has no credentials section", func() {
		It("returns an anonymous resolver without panicking", func() {
			pm, err := ocm.GetPluginManager(context.Background(), &ocmgenericspecv1.Config{})
			Expect(err).ToNot(HaveOccurred())

			resolver, err := ocm.GetCredentialGraph(context.Background(), pm, &ocmgenericspecv1.Config{})

			Expect(err).ToNot(HaveOccurred())
			Expect(resolver).ToNot(BeNil())
		})
	})
})
