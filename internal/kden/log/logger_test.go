package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("logger node", func() {

	DescribeTable("when Logger is configured with JSON handler",
		func(level slog.Level, logFunc func(format string, args ...any), expectedLevel string) {
			buf := &bytes.Buffer{}

			handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{
				Level: level,
			})

			InitLogger(handler)
			logFunc("hello %s", "world")

			var payload map[string]any
			Expect(json.Unmarshal(buf.Bytes(), &payload)).To(Succeed())

			Expect(payload["msg"]).To(Equal("hello world"))
			Expect(payload["level"]).To(Equal(expectedLevel))
		},
		Entry("with debug level", slog.LevelDebug, Debugf, "DEBUG"),
		Entry("with info level", slog.LevelInfo, Infof, "INFO"),
		Entry("with error level", slog.LevelError, Errorf, "ERROR"),
	)

	DescribeTable("when Logger is configured with Text handler",
		func(level slog.Level, logFunc func(format string, args ...any), expectedLevel string) {
			buf := &bytes.Buffer{}

			handler := slog.NewTextHandler(buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})

			InitLogger(handler)
			logFunc("hello %s", "world")

			output := buf.String()

			Expect(output).To(ContainSubstring(fmt.Sprintf("level=%s", expectedLevel)))
			Expect(output).To(ContainSubstring("msg=\"hello world\""))
		},
		Entry("with debug level", slog.LevelDebug, Debugf, "DEBUG"),
		Entry("with info level", slog.LevelInfo, Infof, "INFO"),
		Entry("with error level", slog.LevelError, Errorf, "ERROR"),
	)

	DescribeTable("when Logger is configured with Pretty handler",
		func(level slog.Level, logFunc func(format string, args ...any), expectedLevel string) {
			buf := &bytes.Buffer{}

			handler := NewPrettyLogHandler(buf, &PrettyOptions{
				Level: slog.LevelDebug,
			})

			InitLogger(handler)
			logFunc("hello %s", "world")

			output := buf.String()

			Expect(output).To(ContainSubstring(expectedLevel))
			Expect(output).To(ContainSubstring("hello world"))
		},
		Entry("with debug level", slog.LevelDebug, Debugf, "DEBUG"),
		Entry("with info level", slog.LevelInfo, Infof, "INFO"),
		Entry("with error level", slog.LevelError, Errorf, "ERROR"),
	)

})
