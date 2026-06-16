package validation_test

import (
	"errors"

	"github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"

	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func noErrors(_ []byte, _ string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	return nil, nil
}

func alwaysErrors(_ []byte, _ string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	return nil, errors.New("validation exploded")
}

func schemaErrors(_ []byte, _ string, _ map[string]bool) ([]output.SchemaValidationError, error) {
	return []output.SchemaValidationError{{Path: "/type", Message: "missing type"}}, nil
}

func writeFile(t GinkgoTInterface, name string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	Expect(os.WriteFile(p, []byte("content: true"), 0o600)).To(Succeed())
	return p
}

var _ = Describe("RunValidate", func() {

	BeforeEach(func() {
		config.Config.Output = "json"
	})

	Context("when no files are provided", func() {
		It("falls back to the default file name and returns an error when it does not exist", func() {
			cfg := validation.ValidateConfig{
				DefaultFile:         "nonexistent-default-file",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          noErrors,
			}
			err := validation.RunValidate(nil, cfg)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when a file does not exist", func() {
		It("returns a read error", func() {
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          noErrors,
			}
			err := validation.RunValidate([]string{"/nonexistent/path/file.yaml"}, cfg)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the validate function returns an error", func() {
		It("wraps and returns the error with the component identifier and file path", func() {
			p := writeFile(GinkgoT(), "input.yaml")
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          alwaysErrors,
			}
			err := validation.RunValidate([]string{p}, cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("artifact"))
			Expect(err.Error()).To(ContainSubstring(p))
			Expect(err.Error()).To(ContainSubstring("validation exploded"))
		})
	})

	Context("when all files pass validation", func() {
		It("returns nil", func() {
			p := writeFile(GinkgoT(), "input.yaml")
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          noErrors,
			}
			err := validation.RunValidate([]string{p}, cfg)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when all files with duplicates pass validation", func() {
		It("returns nil", func() {
			p := writeFile(GinkgoT(), "input.yaml")
			pcopy := writeFile(GinkgoT(), "input.yaml")
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          noErrors,
			}
			err := validation.RunValidate([]string{p, pcopy}, cfg)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when files have schema validation errors", func() {
		It("formats and prints them without returning an error", func() {
			p := writeFile(GinkgoT(), "input.yaml")
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          schemaErrors,
			}
			err := validation.RunValidate([]string{p}, cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error when output format is invalid", func() {
			config.Config.Output = "xml"
			p := writeFile(GinkgoT(), "input.yaml")
			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          schemaErrors,
			}
			err := validation.RunValidate([]string{p}, cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve output format for validate command"))
		})
	})

	Context("when multiple files are provided", func() {
		It("accumulates schema errors from all files", func() {
			config.Config.Output = "json"
			t := GinkgoT()
			p1 := writeFile(t, "first.yaml")
			p2 := writeFile(t, "second.yaml")

			callCount := 0
			validateFn := func(_ []byte, _ string, _ map[string]bool) ([]output.SchemaValidationError, error) {
				callCount++
				return []output.SchemaValidationError{{Path: "/type", Message: "missing type"}}, nil
			}

			cfg := validation.ValidateConfig{
				DefaultFile:         "artifact",
				ComponentIdentifier: "artifact",
				CmdDisplayName:      "validate",
				ValidateFn:          validateFn,
			}
			err := validation.RunValidate([]string{p1, p2}, cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(callCount).To(Equal(2))
		})
	})
})
