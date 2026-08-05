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
			Server: config.ServerConfig{
				Addr: ":8090", ReadTimeout: "10s", WriteTimeout: "10s", ShutdownTimeout: "15s", LogLevel: "info",
			},
			OIDC: config.OIDCConfig{
				Enabled:   true,
				IssuerURL: "http://localhost:5556/oauth", ClientID: "konfidence", ClientSecret: "secret",
				Scopes: "openid,customScope", RedirectURL: "http://localhost:8090/api/v1/auth/callback",
				PKCEEnabled: true, StateExpiration: "15m",
			},
			Session: config.SessionConfig{
				Cookie: config.SessionCookieConfig{Name: "kden-session", HTTPOnly: true, SameSite: "SameSiteStrictMode"},
				Expiry: "12h",
			},
			Database: config.DatabaseConfig{Connection: "dbConn"},
		}
	}

	It("accepts a fully valid config and returns parsed durations", func() {
		parsed, err := valid().Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.Server.Addr).To(Equal(":8090"))
		Expect(parsed.OIDC.IssuerURL).To(Equal("http://localhost:5556/oauth"))
		Expect(parsed.OIDC.Enabled).To(BeTrue())
		Expect(parsed.OIDC.ClientID).To(Equal("konfidence"))
		Expect(parsed.OIDC.ClientSecret).To(Equal("secret"))
		Expect(parsed.OIDC.Scopes).To(ConsistOf([]string{"openid", "profile", "customScope"}))
		Expect(parsed.OIDC.RedirectURL).To(Equal("http://localhost:8090/api/v1/auth/callback"))
		Expect(parsed.OIDC.PKCEEnabled).To(BeTrue())
		Expect(parsed.OIDC.StateExpiration.Minutes()).To(Equal(15.0))
		Expect(parsed.Session.Cookie.Name).To(Equal("kden-session"))
		Expect(parsed.Session.Cookie.HTTPOnly).To(BeTrue())
		Expect(parsed.Session.Cookie.Secure).To(BeFalse())
		Expect(parsed.Session.Cookie.SameSite).To(Equal("SameSiteStrictMode"))
		Expect(parsed.Server.ReadTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.Server.WriteTimeout.Seconds()).To(Equal(10.0))
		Expect(parsed.Server.ShutdownTimeout.Seconds()).To(Equal(15.0))
		Expect(parsed.Session.Expiry.Hours()).To(Equal(12.0))
		Expect(parsed.Database.Connection).To(Equal("dbConn"))
	})

	It("rejects an empty addr", func() {
		c := valid()
		c.Server.Addr = ""
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
		Entry("read-timeout", "read-timeout", func(c *config.Config) { c.Server.ReadTimeout = invalidDuration }),
		Entry("write-timeout", "write-timeout", func(c *config.Config) { c.Server.WriteTimeout = invalidDuration }),
		Entry("shutdown-timeout", "shutdown-timeout", func(c *config.Config) { c.Server.ShutdownTimeout = invalidDuration }),
		Entry("oidc-state-expiration", "oidc-state-expiration", func(c *config.Config) { c.OIDC.StateExpiration = invalidDuration }),
		Entry("session-expiry", "session-expiry", func(c *config.Config) { c.Session.Expiry = invalidDuration }),
	)

	It("rejects an unknown log level", func() {
		c := valid()
		c.Server.LogLevel = "verbose"
		_, err := c.Validate()
		Expect(err).To(MatchError(ContainSubstring("log-level")))
	})

	It("allows OIDC provider settings to be omitted when OIDC is disabled", func() {
		c := valid()
		c.OIDC.Enabled = false
		c.OIDC.IssuerURL = ""
		c.OIDC.ClientID = ""
		c.OIDC.ClientSecret = ""
		c.OIDC.RedirectURL = ""

		parsed, err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.OIDC.Enabled).To(BeFalse())
	})

	DescribeTable("rejects empty oidc and session config",
		func(field string, mutate func(*config.Config)) {
			c := valid()
			mutate(&c)
			_, err := c.Validate()
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("oidc-issuer-url", "oidc-issuer-url", func(c *config.Config) { c.OIDC.IssuerURL = "" }),
		Entry("oidc-client-id", "oidc-client-id", func(c *config.Config) { c.OIDC.ClientID = "" }),
		Entry("oidc-client-secret", "oidc-client-secret", func(c *config.Config) { c.OIDC.ClientSecret = "" }),
		Entry("oidc-redirect-url", "oidc-redirect-url", func(c *config.Config) { c.OIDC.RedirectURL = "" }),
		Entry("session-cookie-name", "session-cookie-name", func(c *config.Config) { c.Session.Cookie.Name = "" }),
	)

	It("returns parsed durations on success", func() {
		c := config.Config{
			Server: config.ServerConfig{Addr: ":8090", ReadTimeout: "5s", WriteTimeout: "7s", ShutdownTimeout: "20s", LogLevel: "debug"},
			OIDC: config.OIDCConfig{
				Enabled:   true,
				IssuerURL: "https://idp.example.com", TokenURL: "https://idp.example.com/token",
				AuthorizationURL: "https://idp.example.com/auth", DeviceAuthURL: "https://idp.example.com/device",
				UserInfoURL: "https://idp.example.com/userinfo", JWKSURL: "https://idp.example.com/jwks",
				ClientID: "konfidence", ClientSecret: "secret", RedirectURL: "https://konfidence.example.com/auth/callback",
				StateExpiration: "5m",
			},
			Session: config.SessionConfig{
				Cookie: config.SessionCookieConfig{Name: "custom-session", Secure: true, SameSite: "SameSiteNoneMode"},
				Expiry: "1h",
			},
		}
		parsed, err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.Server.ReadTimeout.Seconds()).To(Equal(5.0))
		Expect(parsed.Server.WriteTimeout.Seconds()).To(Equal(7.0))
		Expect(parsed.Server.ShutdownTimeout.Seconds()).To(Equal(20.0))
		Expect(parsed.OIDC.PKCEEnabled).To(BeFalse())
		Expect(parsed.OIDC.TokenURL).To(Equal("https://idp.example.com/token"))
		Expect(parsed.OIDC.AuthorizationURL).To(Equal("https://idp.example.com/auth"))
		Expect(parsed.OIDC.DeviceAuthURL).To(Equal("https://idp.example.com/device"))
		Expect(parsed.OIDC.UserInfoURL).To(Equal("https://idp.example.com/userinfo"))
		Expect(parsed.OIDC.JWKSURL).To(Equal("https://idp.example.com/jwks"))
		Expect(parsed.OIDC.StateExpiration.Minutes()).To(Equal(5.0))
		Expect(parsed.Session.Cookie.Name).To(Equal("custom-session"))
		Expect(parsed.Session.Cookie.HTTPOnly).To(BeFalse())
		Expect(parsed.Session.Cookie.Secure).To(BeTrue())
		Expect(parsed.Session.Cookie.SameSite).To(Equal("SameSiteNoneMode"))
		Expect(parsed.Session.Expiry.Hours()).To(Equal(1.0))
	})
})
