package pretty_test

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetModelFuncMap", func() {

	Describe("GetModelFuncMap", func() {
		It("should contain the validate key", func() {
			m := pretty.GetModelFuncMap()
			Expect(m).NotTo(BeNil())
			Expect(m).To(HaveKey("validate"))
		})
	})

	Describe("validate model function", func() {
		Context("when called with a valid []SchemaValidationError", func() {
			It("should return TableData with correct columns", func() {
				fn := pretty.GetModelFuncMap()["validate"]
				result := fn([]output.SchemaValidationError{})

				Expect(result).NotTo(BeNil())
				Expect(result.Err).ToNot(HaveOccurred())
				Expect(result.Columns).To(HaveLen(3))
				Expect(result.Columns[0].Title).To(Equal("File"))
				Expect(result.Columns[1].Title).To(Equal("Path"))
				Expect(result.Columns[2].Title).To(Equal("Message"))
			})

			It("should return TableData with one row per error", func() {
				fn := pretty.GetModelFuncMap()["validate"]
				errors := []output.SchemaValidationError{
					{File: "a.yaml", Path: "/foo", Message: "required"},
					{File: "b.yaml", Path: "/bar", Message: "invalid type"},
				}
				result := fn(errors)

				Expect(result.Err).ToNot(HaveOccurred())
				Expect(result.Rows).To(HaveLen(2))
				Expect(result.Rows[0]).To(Equal(table.Row{"a.yaml", "/foo", "required"}))
				Expect(result.Rows[1]).To(Equal(table.Row{"b.yaml", "/bar", "invalid type"}))
			})
		})

		Context("when called with the wrong type", func() {
			It("should return TableData with a descriptive error", func() {
				fn := pretty.GetModelFuncMap()["validate"]
				result := fn("not a slice")

				Expect(result).NotTo(BeNil())
				Expect(result.Err).To(HaveOccurred())
				Expect(result.Err.Error()).To(ContainSubstring("error while creating table for command validate"))
				fmt.Println(result.Err.Error())
			})
		})
	})
})
