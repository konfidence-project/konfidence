package log

import (
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("printer node", func() {

	DescribeTable("when ResolveLogHandler should work correctly",
		func(level, format string) {
			handler, err := ResolveLogHandler(level, format)

			Expect(handler).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("with valid JSON format", "info", jsonLogFormat),
		Entry("with valid Text format", "debug", textLogFormat),
		Entry("with valid Pretty format", "info", prettyLogFormat),
	)

	DescribeTable("when ResolveLogHandler should throw an error",
		func(level, format, errorMessage string) {
			handler, err := ResolveLogHandler(level, format)

			Expect(handler).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorMessage))
		},
		Entry("with invalid format configuration", "info", "xml", "invalid log format provided: xml"),
		Entry("with invalid level configuration", "invalid", jsonLogFormat, "invalid log level provided: invalid"),
	)

	DescribeTable("when toLogLevel should work correctly",
		func(providedLevel string, expectedLevel slog.Level) {
			level, err := toLogLevel(providedLevel)

			Expect(err).NotTo(HaveOccurred())
			Expect(level).To(Equal(expectedLevel))
		},
		Entry("with valid DEBUG format", "debug", slog.LevelDebug),
		Entry("with valid INFO format", "info", slog.LevelInfo),
	)

	Context("when toLogLevel should throw an error", func() {
		It("should return an error for invalid levels", func() {
			level, err := toLogLevel("invalid")

			Expect(level).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("invalid log level provided: invalid"))
		})
	})

})
