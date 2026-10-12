package sanitize_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/sanitize"
)

var _ = Describe("Kubernetes name sanitization", func() {
	DescribeTable("sanitizes DNS label names",
		func(input, expected string) {
			Expect(sanitize.DNSLabelName(input)).To(Equal(expected))
		},
		Entry("simple valid name", "my-resource", "my-resource"),
		Entry("uppercase letters", "MyResource", "myresource"),
		Entry("special characters", "my_resource@example.com", "my-resource-example-com"),
		Entry("leading and trailing invalid chars", "_my-resource_", "my-resource"),
		Entry("name too long",
			"this-is-a-very-long-name-that-exceeds-sixty-three-characters-limit",
			"this-is-a-very-long-name-that-exceeds-sixty-three-characters-li",
		),
		Entry("empty string", "", ""),
		Entry("multiple invalid characters", "a@#b$%c", "a--b--c"),
		Entry("only invalid characters", "@#$%^&*()", ""),
	)

	DescribeTable("sanitizes DNS subdomain names",
		func(input, expected string) {
			Expect(sanitize.DNSSubdomainName(input)).To(Equal(expected))
		},
		Entry("simple valid name with dashes", "my-resource", "my-resource"),
		Entry("simple valid name with dots", "test.resource.com", "test.resource.com"),
		Entry("simple valid name with dashes and dots", "test.resource-123.com", "test.resource-123.com"),
		Entry("uppercase letters", "MyResource", "myresource"),
		Entry("special characters", "my_resource@example.com", "my-resource-example.com"),
		Entry("leading and trailing invalid chars", "-my-resource_", "my-resource"),
		Entry("name too long",
			"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit",
			"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit.this-is-a-very-lo",
		),
		Entry("empty string", "", ""),
		Entry("multiple invalid characters", "a@#b$%c", "a--b--c"),
		Entry("only invalid characters", "@#$%^&*()", ""),
	)

	DescribeTable("sanitizes resource names",
		func(input, expected string) {
			Expect(sanitize.ResourceName(input)).To(Equal(expected))
		},
		Entry("simple valid name with dashes", "my-resource", "my-resource"),
		Entry("simple valid name with dots", "test.resource.com", "test-resource-com"),
		Entry("simple valid name with dashes and dots", "test.resource-123.com", "test-resource-123-com"),
		Entry("special characters", "my_resource@example.com", "my-resource-example-com"),
		Entry("leading and trailing invalid chars", "-my-resource_", "my-resource"),
		Entry("name too long",
			"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit."+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit",
			"this-is-a-very-long-name-that-exceeds-253-characters-limit-"+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-"+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-"+
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-this-is-a-very-lo",
		),
		Entry("empty string", "", ""),
		Entry("multiple invalid characters", "a@#b$%c", "a--b--c"),
		Entry("only invalid characters", "@#$%^&*()", ""),
	)
})
