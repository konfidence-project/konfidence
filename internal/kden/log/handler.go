package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	tabSeparator     = '\t'
	newLineSeparator = "\n"
)

type PrettyLogHandler struct {
	opts           PrettyOptions
	preformatted   []byte   // data from WithGroup and WithAttrs
	unopenedGroups []string // groups from WithGroup that haven't been opened
	mu             *sync.Mutex
	out            io.Writer
}

type PrettyOptions struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
}

func NewPrettyLogHandler(out io.Writer, opts *PrettyOptions) *PrettyLogHandler {
	handler := &PrettyLogHandler{
		out: out,
		mu:  &sync.Mutex{},
	}
	if opts != nil {
		handler.opts = *opts
	}
	if handler.opts.Level == nil {
		handler.opts.Level = slog.LevelInfo
	}
	return handler
}

// Enabled checks the logs before it processes any of its arguments,
// to see if it should proceed, based on the logging level.
func (h *PrettyLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups, implemented with pre-formatting.
func (h *PrettyLogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.unopenedGroups = make([]string, len(h.unopenedGroups)+1)
	copy(h2.unopenedGroups, h.unopenedGroups)
	h2.unopenedGroups[len(h2.unopenedGroups)-1] = name
	return &h2
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments, implemented with pre-formatting.
func (h *PrettyLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h

	pre := slices.Clip(h.preformatted)
	h2.preformatted = h2.appendUnopenedGroups(pre)
	h2.unopenedGroups = nil

	for _, a := range attrs {
		h2.preformatted = h2.appendAttr(h2.preformatted, a)
	}
	return &h2
}

func (h *PrettyLogHandler) appendUnopenedGroups(buf []byte) []byte {
	for _, g := range h.unopenedGroups {
		buf = fmt.Appendf(buf, " %s:\t", g)
	}
	return buf
}

// Handle processes all the details, attached to a Record,
// to be logged for a single invocation of a Logger's output method.
func (h *PrettyLogHandler) Handle(_ context.Context, r slog.Record) error {
	bufp := allocBuf()
	buf := *bufp
	defer func() {
		*bufp = buf
		freeBuf(bufp)
	}()

	buf = h.handle(buf, r)

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, err := h.out.Write(buf); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

func (h *PrettyLogHandler) handle(buf []byte, record slog.Record) []byte {
	colorSchema := createColorSchema(record.Level)

	// Timestamp
	if !record.Time.IsZero() {
		buf = append(buf, record.Time.Format(time.RFC3339)...)
		buf = append(buf, tabSeparator)
	}

	// Level
	buf = append(buf, colorSchema.Sprint(record.Level.String())...)
	buf = append(buf, tabSeparator)

	// Message
	buf = append(buf, record.Message...)
	buf = append(buf, tabSeparator)

	// Insert preformatted attributes just after built-in ones.
	buf = append(buf, h.preformatted...)
	if record.NumAttrs() > 0 {
		buf = h.appendUnopenedGroups(buf)
		record.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a)
			return true
		})
	}
	buf = append(buf, newLineSeparator...)

	return buf
}

func createColorSchema(level slog.Level) *color.Color {
	colorSchema := color.New()
	switch level {
	case slog.LevelDebug:
		colorSchema = colorSchema.Add(color.FgHiBlack)
	case slog.LevelInfo:
		colorSchema = colorSchema.Add(color.FgHiBlue)
	case slog.LevelError:
		colorSchema = colorSchema.Add(color.FgRed)
	}
	return colorSchema
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func allocBuf() *[]byte {
	return bufPool.Get().(*[]byte)
}

func freeBuf(b *[]byte) {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

func (h *PrettyLogHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	switch a.Value.Kind() {
	case slog.KindString:
		buf = fmt.Appendf(buf, "%s: %q\t", a.Key, a.Value.String())
	case slog.KindTime:
		buf = fmt.Appendf(buf, "%s: %s\t", a.Key, a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s:\t", a.Key)
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, "%s: %s\t", a.Key, a.Value)
	}
	return buf
}
