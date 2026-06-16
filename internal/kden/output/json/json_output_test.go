package json

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("json format output", func() {
	Describe("JSONFormat", func() {
		Context("when the mapped json input data is passed", func() {
			It("should format in json", func() {
				converted := &JSONConverter{}
				object, _ := converted.ToMap([]byte("{\"test\": \"test\", \"tested\": \"tested\"}"))
				formatted := &JSONFormatter{}
				_, err := formatted.Format(object)

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("JSONToMap", func() {
		Context("when encoded json object is passed", func() {
			It("should return map result", func() {
				converted := &JSONConverter{}
				_, err := converted.ToMap([]byte("{\"test\": \"test\", \"tested\": \"tested\"}"))

				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when the invalid input data is passed", func() {
			It("should throw an error when converting to map", func() {
				converted := &JSONConverter{}
				_, err := converted.ToMap([]byte("{\"test\": \"t\"est\", \"tested\": \"tested\"}"))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("error occurred during parse of object to map: " +
					"{\"test\": \"t\"est\", \"tested\": \"tested\"} : yaml: did not find expected ',' or '}'"))
			})
		})
	})
})
