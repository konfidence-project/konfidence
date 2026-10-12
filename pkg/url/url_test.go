package url_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	urlutil "github.com/konfidence-project/konfidence/pkg/url"
)

var _ = Describe("URL utilities", func() {
	DescribeTable("extracts hostnames",
		func(input, expected string) {
			result, _ := urlutil.ExtractHostname(input)
			Expect(result).To(Equal(expected))
		},
		Entry("full URL", "https://user@test.registry.com:5100/v2/ocm/repository", "test.registry.com"),
		Entry("simple host with www prefix and no port", "www.registry.com", "registry.com"),
		Entry("invalid URL", "?www.registry.com", ""),
		Entry("URL with dashes and query parameter", "http://registry-ocm.test/test?id=123", "registry-ocm.test"),
	)

	DescribeTable("extracts hostnames with optional ports",
		func(input, expected string) {
			result, _ := urlutil.ExtractHostnameWithOptionalPort(input)
			Expect(result).To(Equal(expected))
		},
		Entry("full URL", "https://user@test.registry.com:5100/v2/ocm/repository", "test.registry.com:5100"),
		Entry("simple host with www prefix and no port", "www.registry.com", "registry.com"),
		Entry("invalid URL", "?www.registry.com", ""),
		Entry("URL with dashes and query parameter", "http://registry-ocm.test:8080/test?id=123", "registry-ocm.test:8080"),
	)
})
