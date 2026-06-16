package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handler node", func() {

	Describe("NewPrettyLogHandler", func() {
		It("should initialize with the provided options", func() {
			opts := &PrettyOptions{Level: slog.LevelDebug}
			handler := NewPrettyLogHandler(os.Stdout, opts)

			Expect(handler).NotTo(BeNil())
			Expect(handler.out).To(Equal(os.Stdout))
			Expect(handler.opts.Level).To(Equal(slog.LevelDebug))
		})

		Context("when options are nil", func() {
			It("should initialize with default options", func() {
				handler := NewPrettyLogHandler(os.Stdout, nil)

				Expect(handler).NotTo(BeNil())
				Expect(handler.out).To(Equal(os.Stdout))
				Expect(handler.opts.Level).To(Equal(slog.LevelInfo))
			})
		})
	})

	Describe("Enabled", func() {
		Context("when the log level is greater than or equal to the handler's level", func() {
			It("should return true", func() {
				handler := &PrettyLogHandler{opts: PrettyOptions{Level: slog.LevelInfo}}
				Expect(handler.Enabled(context.Background(), slog.LevelInfo)).To(BeTrue())
				Expect(handler.Enabled(context.Background(), slog.LevelError)).To(BeTrue())
			})
		})

		Context("when the log level is less than the handler's level", func() {
			It("should return false", func() {
				handler := &PrettyLogHandler{opts: PrettyOptions{Level: slog.LevelInfo}}
				Expect(handler.Enabled(context.Background(), slog.LevelDebug)).To(BeFalse())
			})
		})
	})

	Describe("WithGroup", func() {
		Context("when the group name is empty", func() {
			It("should return the same handler", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{},
				}
				result := handler.WithGroup("")
				Expect(result).To(BeIdenticalTo(handler))
			})
		})

		Context("when the group name is not empty", func() {
			It("should return a new handler with the group added", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{},
				}
				result := handler.WithGroup("group1")
				Expect(result).NotTo(BeIdenticalTo(handler))

				newHandler, ok := result.(*PrettyLogHandler)
				Expect(ok).To(BeTrue())
				Expect(newHandler.unopenedGroups).To(Equal([]string{"group1"}))
				Expect(handler.unopenedGroups).To(BeEmpty()) // Ensure original handler is unchanged
			})

			It("should preserve existing groups and add the new group", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{},
				}
				handler.unopenedGroups = []string{"group1"}
				result := handler.WithGroup("group2")

				newHandler, ok := result.(*PrettyLogHandler)
				Expect(ok).To(BeTrue())
				Expect(newHandler.unopenedGroups).To(Equal([]string{"group1", "group2"}))
				Expect(handler.unopenedGroups).To(Equal([]string{"group1"})) // Ensure original handler is unchanged
			})
		})

	})

	Describe("WithAttrs", func() {
		Context("when no attributes are provided", func() {
			It("should return the same handler", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{"group1", "group2"},
					preformatted:   []byte("initial:"),
				}
				result := handler.WithAttrs(nil)
				Expect(result).To(BeIdenticalTo(handler))
			})
		})

		Context("when attributes are provided", func() {
			It("should return a new handler with attributes appended", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{"group1", "group2"},
					preformatted:   []byte("initial:"),
				}
				attrs := make([]slog.Attr, 0, 2)
				attrs = append(attrs, slog.String("key1", "value1"))
				attrs = append(attrs, slog.String("key2", "value2"))

				result := handler.WithAttrs(attrs)

				Expect(result).NotTo(BeIdenticalTo(handler))

				newHandler, ok := result.(*PrettyLogHandler)

				expectedFields := []string{
					"initial:", "group1:", "group2:", "key1:", "\"value1\"", "key2:", "\"value2\"",
				}

				Expect(ok).To(BeTrue())
				Expect(strings.Fields(string(newHandler.preformatted))).To(Equal(expectedFields))
				Expect(handler.unopenedGroups).To(Equal([]string{"group1", "group2"})) // Ensure original handler is unchanged
			})

			It("should clear unopened groups in the new handler", func() {
				handler := &PrettyLogHandler{
					unopenedGroups: []string{"group1", "group2"},
					preformatted:   []byte("initial:"),
				}
				attrs := make([]slog.Attr, 0, 1)
				attrs = append(attrs, slog.String("key1", "value1"))

				result := handler.WithAttrs(attrs)

				newHandler, ok := result.(*PrettyLogHandler)
				Expect(ok).To(BeTrue())
				Expect(newHandler.unopenedGroups).To(BeNil())
			})
		})
	})

	Describe("handle", func() {
		Context("when the record has a timestamp, level, and output", func() {
			It("should format the log correctly", func() {
				handler := PrettyLogHandler{
					preformatted: []byte("preformatted:"),
				}
				buf := []byte{}
				record := slog.Record{
					Time:    time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
					Level:   slog.LevelInfo,
					Message: "Test output",
				}
				result := handler.handle(buf, record)

				expectedFields := []string{"2023-10-01T12:00:00Z", "INFO", "Test", "output", "preformatted:"}

				Expect(strings.Fields(string(result))).To(Equal(expectedFields))
			})
		})

		Context("when the record has attributes", func() {
			It("should include the attributes in the log", func() {
				handler := PrettyLogHandler{
					preformatted:   []byte("preformatted:"),
					unopenedGroups: []string{"group1", "group2"},
				}
				buf := []byte{}
				record := slog.Record{
					Time:    time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
					Level:   slog.LevelInfo,
					Message: "Test output",
				}
				record.AddAttrs(
					slog.Attr{Key: "key1", Value: slog.StringValue("value1")},
					slog.Attr{Key: "key2", Value: slog.StringValue("value2")},
					slog.Attr{Key: "key3", Value: slog.IntValue(100)},
				)
				result := handler.handle(buf, record)

				expectedFields := []string{"2023-10-01T12:00:00Z", "INFO", "Test", "output", "preformatted:",
					"group1:", "group2:", "key1:", "\"value1\"", "key2:", "\"value2\"", "key3:", "100",
				}

				Expect(strings.Fields(string(result))).To(Equal(expectedFields))
			})
		})
	})
})
