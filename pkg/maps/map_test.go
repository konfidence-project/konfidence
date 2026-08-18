package maps_test

import (
	"github.com/konfidence-project/konfidence/pkg/maps"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MapUtils", func() {
	DescribeTable("GetValueFromRawMap when the key exists",
		func(raw []byte, key string, expected interface{}) {
			result, err := maps.GetValueFromRawMap(raw, key)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("string value", []byte(`{"name": "my-component"}`), "name", "my-component"),
		Entry("numeric value", []byte(`{"count": 42}`), "count", float64(42)),
		Entry("boolean value", []byte(`{"enabled": true}`), "enabled", true),
		Entry("nested object value", []byte(`{"meta": {"version": "v1"}}`), "meta", map[string]interface{}{"version": "v1"}),
		Entry("first of multiple keys", []byte(`{"a": "first", "b": "second"}`), "a", "first"),
		Entry("second of multiple keys", []byte(`{"a": "first", "b": "second"}`), "b", "second"),
	)
	Describe("GetValueFromRawMap", func() {
		Context("when the key exists with a null value", func() {
			It("should return nil without error", func() {
				result, err := maps.GetValueFromRawMap([]byte(`{"tag": null}`), "tag")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("when the key does not exist", func() {
			It("should return an error with the missing key name", func() {
				result, err := maps.GetValueFromRawMap([]byte(`{"name": "my-component"}`), "missing")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("missing"))
			})
		})

		Context("when the input is invalid JSON", func() {
			It("should return an error", func() {
				result, err := maps.GetValueFromRawMap([]byte(`not json`), "key")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("when the input is empty", func() {
			It("should return an error", func() {
				result, err := maps.GetValueFromRawMap([]byte(``), "key")

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("CheckIfValueIsPresent", func() {
		It("returns false if value does not exist", func() {
			resourceJsonPaths := map[string]bool{}
			resourceJsonPaths["invalid"] = true
			result := maps.CheckIfValueIsPresent(resourceJsonPaths, "valid")
			Expect(result).To(BeFalse())
		})

		It("returns true if value exists", func() {
			resourceJsonPaths := map[string]bool{}
			resourceJsonPaths["valid"] = true
			result := maps.CheckIfValueIsPresent(resourceJsonPaths, "valid")
			Expect(result).To(BeTrue())
		})

		It("returns false if map is empty", func() {
			resourceJsonPaths := map[string]bool{}
			result := maps.CheckIfValueIsPresent(resourceJsonPaths, "any")
			Expect(result).To(BeFalse())
		})
	})

	Describe("GetDistinctValues", func() {
		It("returns non-duplicate values", func() {
			result := maps.GetDistinctValues([]string{"valid", "valid", "valid2", "valid", "valid2"})
			Expect(result).To(Equal([]string{"valid", "valid2"}))
		})
	})
})
