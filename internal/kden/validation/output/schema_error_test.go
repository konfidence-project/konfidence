package output_test

import (
	"errors"

	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/message"
)

// stubKind is a minimal ErrorKind that returns a fixed output and keyword path.
type stubKind struct {
	msg  string
	path []string
}

func (s *stubKind) KeywordPath() []string { return s.path }
func (s *stubKind) LocalizedString(_ *message.Printer) string {
	return s.msg
}

func newValidationError(keywordLocation []string,
	errKind jsonschema.ErrorKind,
	causes ...*jsonschema.ValidationError) *jsonschema.ValidationError {
	return &jsonschema.ValidationError{
		SchemaURL:        "file:///test.json",
		InstanceLocation: keywordLocation,
		ErrorKind:        errKind,
		Causes:           causes,
	}
}

var _ = Describe("ExtractSchemaValidationErrors", func() {

	const filePath = "component.yaml"

	Describe("when the error is a jsonschema.ValidationError", func() {
		Context("with a single leaf error", func() {
			It("should return one entry with the file and output set", func() {
				err := newValidationError(nil, &kind.Schema{Location: "file:///test.json"},
					newValidationError([]string{"name"}, &stubKind{msg: "missing property", path: []string{"required"}}),
				)

				result, extractErr := output.ExtractSchemaValidationErrors(err, filePath)

				Expect(extractErr).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].File).To(Equal(filePath))
				Expect(result[0].Message).To(Equal("missing property"))
			})
		})

		Context("with multiple leaf errors", func() {
			It("should return an entry for each cause, all with the file set", func() {
				err := newValidationError(nil, &kind.Schema{Location: "file:///test.json"},
					newValidationError([]string{"name"}, &stubKind{msg: "missing name", path: []string{"required"}}),
					newValidationError([]string{"version"}, &stubKind{msg: "missing version", path: []string{"required"}}),
				)

				result, extractErr := output.ExtractSchemaValidationErrors(err, filePath)

				Expect(extractErr).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(2))
				for _, e := range result {
					Expect(e.File).To(Equal(filePath))
					Expect(e.Message).NotTo(BeEmpty())
				}
			})
		})

		Context("with a nested error tree", func() {
			It("should flatten all leaf errors and set the file on each", func() {
				leaf1 := newValidationError([]string{"name"}, &stubKind{msg: "wrong type", path: []string{"type"}})
				leaf2 := newValidationError([]string{"version"}, &stubKind{msg: "missing version", path: []string{"required"}})
				mid := newValidationError(nil, &kind.Group{}, leaf1, leaf2)
				root := newValidationError(nil, &kind.Schema{Location: "file:///test.json"}, mid)

				result, extractErr := output.ExtractSchemaValidationErrors(root, filePath)

				Expect(extractErr).NotTo(HaveOccurred())
				Expect(result).NotTo(BeEmpty())
				for _, e := range result {
					Expect(e.File).To(Equal(filePath))
				}
			})
		})

		Context("with a keyword path on the leaf", func() {
			It("should set the path from the keyword location", func() {
				err := newValidationError(nil, &kind.Schema{Location: "file:///test.json"},
					newValidationError([]string{"count"},
						&stubKind{msg: "expected integer", path: []string{"properties", "count", "type"}}),
				)

				result, extractErr := output.ExtractSchemaValidationErrors(err, filePath)

				Expect(extractErr).NotTo(HaveOccurred())
				Expect(result[0].Path).NotTo(BeEmpty())
			})
		})
	})

	Describe("when the error is not a jsonschema.ValidationError", func() {
		It("should return the original error unchanged", func() {
			err := errors.New("some other error")

			result, extractErr := output.ExtractSchemaValidationErrors(err, filePath)

			Expect(result).To(BeNil())
			Expect(extractErr).To(MatchError(err))
		})
	})
})
