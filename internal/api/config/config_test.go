package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
)

const invalidDuration = "bad"

var _ = Describe("Config.Validate", func() {
	valid := func() config.Config {
		return config.Config{
			Addr:            ":8090",
			ReadTimeout:     "10s",
			WriteTimeout:    "10s",
			ShutdownTimeout: "15s",
			LogLevel:        "info",
			AuthIssuerURL:   "http://localhost:5556/dex",
			AuthClientID:    "konfidence",
			AuthRedirectURL: "http://localhost:8090/api/v1/auth/callback",
		}
	}

	It("accepts a fully valid config and returns parsed durations", func() {
		parsed, err := valid().Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.Addr).To(Equal(":8090"))
		Expect(parsed.AuthIssuerURL).To(Equal("http://localhost:5556/dex"))
		Expect(parsed.AuthClientID).To(Equal("konfidence"))
		Expect(parsed.AuthRedirectURL).To(Equal("http://localhost:8090/api/v1/auth/callback"))
		Expect(parsed.ReadTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.WriteTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.ShutdownTimeout.Seconds()).To(Equal(15.0))
	})

	It("rejects an empty addr", func() {
		c := valid()
		c.Addr = ""
		_, err := c.Validate()
		Expect(err).To(MatchError(ContainSubstring("addr")))
	})

	DescribeTable("rejects invalid durations",
		func(field string, mutate func(*config.Config)) {
			c := valid()
			mutate(&c)
			_, err := c.Validate()
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("read-timeout", "read-timeout", func(c *config.Config) { c.ReadTimeout = invalidDuration }),
		Entry("write-timeout", "write-timeout", func(c *config.Config) { c.WriteTimeout = invalidDuration }),
		Entry("shutdown-timeout", "shutdown-timeout", func(c *config.Config) { c.ShutdownTimeout = invalidDuration }),
	)

	It("rejects an unknown log level", func() {
		c := valid()
		c.LogLevel = "verbose"
		_, err := c.Validate()
		Expect(err).To(MatchError(ContainSubstring("log-level")))
	})

	DescribeTable("rejects empty auth config",
		func(field string, mutate func(*config.Config)) {
			c := valid()
			mutate(&c)
			_, err := c.Validate()
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("auth-issuer-url", "auth-issuer-url", func(c *config.Config) { c.AuthIssuerURL = "" }),
		Entry("auth-client-id", "auth-client-id", func(c *config.Config) { c.AuthClientID = "" }),
		Entry("auth-redirect-url", "auth-redirect-url", func(c *config.Config) { c.AuthRedirectURL = "" }),
	)

	It("returns parsed durations on success", func() {
		c := config.Config{
			Addr:            ":8090",
			ReadTimeout:     "5s",
			WriteTimeout:    "7s",
			ShutdownTimeout: "20s",
			LogLevel:        "debug",
			AuthIssuerURL:   "https://idp.example.com",
			AuthClientID:    "konfidence",
			AuthRedirectURL: "https://konfidence.example.com/auth/callback",
		}
		parsed, err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.ReadTimeout.Seconds()).To(Equal(5.0))
		Expect(parsed.WriteTimeout.Seconds()).To(Equal(7.0))
		Expect(parsed.ShutdownTimeout.Seconds()).To(Equal(20.0))
	})
})
