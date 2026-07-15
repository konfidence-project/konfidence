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
		}
	}

	It("accepts a fully valid config and returns parsed durations", func() {
		parsed, err := valid().Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.Addr).To(Equal(":8090"))
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

	It("returns parsed durations on success", func() {
		c := config.Config{
			Addr:            ":8090",
			ReadTimeout:     "5s",
			WriteTimeout:    "7s",
			ShutdownTimeout: "20s",
			LogLevel:        "debug",
		}
		parsed, err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.ReadTimeout.Seconds()).To(Equal(5.0))
		Expect(parsed.WriteTimeout.Seconds()).To(Equal(7.0))
		Expect(parsed.ShutdownTimeout.Seconds()).To(Equal(20.0))
	})
})
