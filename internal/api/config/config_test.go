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
			Addr:                  ":8090",
			ReadTimeout:           "10s",
			WriteTimeout:          "10s",
			ShutdownTimeout:       "15s",
			LogLevel:              "info",
			OIDCIssuerURL:         "http://localhost:5556/oauth",
			OIDCClientID:          "konfidence",
			OIDCClientSecret:      "secret",
			OIDCScopes:            "openid,customScope",
			OIDCRedirectURL:       "http://localhost:8090/api/v1/auth/callback",
			OIDCPKCEEnabled:       true,
			OIDCStateExpiration:   "15m",
			SessionCookieName:     "kden-session",
			SessionCookieHTTPOnly: true,
			SessionCookieSecure:   false,
			SessionCookieSameSite: "SameSiteStrictMode",
			SessionExpiry:         "12h",
			DBConnection:          "dbConn",
		}
	}

	It("accepts a fully valid config and returns parsed durations", func() {
		parsed, err := valid().Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.Addr).To(Equal(":8090"))
		Expect(parsed.OIDCIssuerURL).To(Equal("http://localhost:5556/oauth"))
		Expect(parsed.OIDCClientID).To(Equal("konfidence"))
		Expect(parsed.OIDCClientSecret).To(Equal("secret"))
		Expect(parsed.OIDCScopes).To(ConsistOf([]string{"openid", "profile", "offline_access", "groups", "customScope"}))
		Expect(parsed.OIDCRedirectURL).To(Equal("http://localhost:8090/api/v1/auth/callback"))
		Expect(parsed.OIDCPKCEEnabled).To(BeTrue())
		Expect(parsed.OIDCStateExpiration.Minutes()).To(Equal(15.0))
		Expect(parsed.SessionCookieName).To(Equal("kden-session"))
		Expect(parsed.SessionCookieHTTPOnly).To(BeTrue())
		Expect(parsed.SessionCookieSecure).To(BeFalse())
		Expect(parsed.SessionCookieSameSite).To(Equal("SameSiteStrictMode"))
		Expect(parsed.ReadTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.WriteTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.ShutdownTimeout.Seconds()).To(Equal(15.0))
		Expect(parsed.SessionExpiry.Hours()).To(Equal(12.0))
		Expect(parsed.DBConnection).To(Equal("dbConn"))
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
		Entry("oidc-state-expiration", "oidc-state-expiration", func(c *config.Config) { c.OIDCStateExpiration = invalidDuration }),
		Entry("session-expiry", "session-expiry", func(c *config.Config) { c.SessionExpiry = invalidDuration }),
	)

	It("rejects an unknown log level", func() {
		c := valid()
		c.LogLevel = "verbose"
		_, err := c.Validate()
		Expect(err).To(MatchError(ContainSubstring("log-level")))
	})

	DescribeTable("rejects empty oidc and session config",
		func(field string, mutate func(*config.Config)) {
			c := valid()
			mutate(&c)
			_, err := c.Validate()
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("oidc-issuer-url", "oidc-issuer-url", func(c *config.Config) { c.OIDCIssuerURL = "" }),
		Entry("oidc-client-id", "oidc-client-id", func(c *config.Config) { c.OIDCClientID = "" }),
		Entry("oidc-client-secret", "oidc-client-secret", func(c *config.Config) { c.OIDCClientSecret = "" }),
		Entry("oidc-redirect-url", "oidc-redirect-url", func(c *config.Config) { c.OIDCRedirectURL = "" }),
		Entry("session-cookie-name", "session-cookie-name", func(c *config.Config) { c.SessionCookieName = "" }),
	)

	It("returns parsed durations on success", func() {
		c := config.Config{
			Addr:                  ":8090",
			ReadTimeout:           "5s",
			WriteTimeout:          "7s",
			ShutdownTimeout:       "20s",
			LogLevel:              "debug",
			OIDCIssuerURL:         "https://idp.example.com",
			OIDCTokenURL:          "https://idp.example.com/token",
			OIDCAuthorizationURL:  "https://idp.example.com/auth",
			OIDCDeviceAuthURL:     "https://idp.example.com/device",
			OIDCUserInfoURL:       "https://idp.example.com/userinfo",
			OIDCJWKSURL:           "https://idp.example.com/jwks",
			OIDCClientID:          "konfidence",
			OIDCClientSecret:      "secret",
			OIDCRedirectURL:       "https://konfidence.example.com/auth/callback",
			OIDCPKCEEnabled:       false,
			OIDCStateExpiration:   "5m",
			SessionCookieName:     "custom-session",
			SessionCookieHTTPOnly: false,
			SessionCookieSecure:   true,
			SessionCookieSameSite: "SameSiteNoneMode",
			SessionExpiry:         "1h",
		}
		parsed, err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.ReadTimeout.Seconds()).To(Equal(5.0))
		Expect(parsed.WriteTimeout.Seconds()).To(Equal(7.0))
		Expect(parsed.ShutdownTimeout.Seconds()).To(Equal(20.0))
		Expect(parsed.OIDCPKCEEnabled).To(BeFalse())
		Expect(parsed.OIDCTokenURL).To(Equal("https://idp.example.com/token"))
		Expect(parsed.OIDCAuthorizationURL).To(Equal("https://idp.example.com/auth"))
		Expect(parsed.OIDCDeviceAuthURL).To(Equal("https://idp.example.com/device"))
		Expect(parsed.OIDCUserInfoURL).To(Equal("https://idp.example.com/userinfo"))
		Expect(parsed.OIDCJWKSURL).To(Equal("https://idp.example.com/jwks"))
		Expect(parsed.OIDCStateExpiration.Minutes()).To(Equal(5.0))
		Expect(parsed.SessionCookieName).To(Equal("custom-session"))
		Expect(parsed.SessionCookieHTTPOnly).To(BeFalse())
		Expect(parsed.SessionCookieSecure).To(BeTrue())
		Expect(parsed.SessionCookieSameSite).To(Equal("SameSiteNoneMode"))
		Expect(parsed.SessionExpiry.Hours()).To(Equal(1.0))
	})
})
