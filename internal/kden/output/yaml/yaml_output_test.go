package yaml

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("yaml format output", func() {
	Describe("YAMLFormat", func() {
		Context("when the mapped yaml input data is passed", func() {
			It("should format in yaml", func() {
				converted := &YAMLConverter{}
				object, _ := converted.ToMap([]byte("test: test"))
				formatted := &YAMLFormatter{}
				_, err := formatted.Format(object)

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("YAMLToMap", func() {
		Context("when encoded yaml object is passed", func() {
			It("should return map result", func() {
				converted := &YAMLConverter{}
				_, err := converted.ToMap([]byte("test: test"))

				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when the invalid input data is passed", func() {
			It("should throw an error when converting to map", func() {
				converted := &YAMLConverter{}
				_, err := converted.ToMap([]byte("{\"test\": \"t\"est\"}"))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("error occurred during parse of object to map: " +
					"{\"test\": \"t\"est\"} : yaml: did not find expected ',' or '}'"))
			})
		})
	})
})
