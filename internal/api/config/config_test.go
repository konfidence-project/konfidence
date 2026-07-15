package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
)

var _ = Describe("Config.Validate", func() {
	valid := func() config.Config {
		return config.Config{
			Addr:            ":8090",
			ReadTimeout:     "10s",
			WriteTimeout:    "10s",
			ShutdownTimeout: "15s",
			LogLevel:        "info",
		}
	}

	It("accepts a fully valid config", func() {
		Expect(valid().Validate()).To(Succeed())
	})

	It("rejects an empty addr", func() {
		c := valid()
		c.Addr = ""
		Expect(c.Validate()).To(MatchError(ContainSubstring("addr")))
	})

	DescribeTable("rejects invalid durations",
		func(field string, mutate func(*config.Config)) {
			c := valid()
			mutate(&c)
			Expect(c.Validate()).To(MatchError(ContainSubstring(field)))
		},
		Entry("read-timeout", "read-timeout", func(c *config.Config) { c.ReadTimeout = "bad" }),
		Entry("write-timeout", "write-timeout", func(c *config.Config) { c.WriteTimeout = "bad" }),
		Entry("shutdown-timeout", "shutdown-timeout", func(c *config.Config) { c.ShutdownTimeout = "bad" }),
	)

	It("rejects an unknown log level", func() {
		c := valid()
		c.LogLevel = "verbose"
		Expect(c.Validate()).To(MatchError(ContainSubstring("log-level")))
	})
})

var _ = Describe("Config.Parse", func() {
	It("returns parsed durations", func() {
		c := config.Config{
			Addr:            ":8090",
			ReadTimeout:     "5s",
			WriteTimeout:    "7s",
			ShutdownTimeout: "20s",
			LogLevel:        "debug",
		}
		p := c.Parse()
		Expect(p.ReadTimeout.Seconds()).To(Equal(5.0))
		Expect(p.WriteTimeout.Seconds()).To(Equal(7.0))
		Expect(p.ShutdownTimeout.Seconds()).To(Equal(20.0))
	})
})
