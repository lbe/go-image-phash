// Package consoleslog provides a minimal console slog handler: it renders
// each record as one human-readable line of the form
//
//	2009-11-10T23:00:00Z INFO message key=value
//
// so commands like cmd/phash get readable logs from the standard library's
// log/slog without pulling in a third-party handler.
package consoleslog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"
)

// timeFormat is the timestamp layout every rendered line and time-valued
// attribute uses.
const timeFormat = time.RFC3339

// Handler is a slog.Handler that writes human-readable lines to its writer.
type Handler struct {
	mu     *sync.Mutex // shared by derived handlers so writes to w stay serialized
	w      io.Writer
	level  slog.Level
	prefix string         // group names accumulated by WithGroup, joined with "."
	attrs  []prefixedAttr // attrs added by WithAttrs, with the prefix at add time
}

// prefixedAttr is an attribute together with the group prefix its key had
// when WithAttrs added it, so later WithGroup calls do not re-prefix it.
type prefixedAttr struct {
	prefix string
	attr   slog.Attr
}

// New returns a console Handler writing to w at Info level.
func New(w io.Writer) *Handler {
	return &Handler{mu: new(sync.Mutex), w: w, level: slog.LevelInfo}
}

// NewLogger returns a slog.Logger backed by a console Handler writing to
// standard error.
func NewLogger() *slog.Logger {
	return slog.New(New(os.Stderr))
}

// Enabled reports whether the record's level passes the handler threshold.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle renders one record as a single console line: time, level, message,
// then each attribute as key=value.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	line := h.render(r)

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(line)
	return err
}

// render formats one record as a single console line — time, level, message,
// then each attribute as key=value — without touching the shared writer, so
// Handle stays a lock-around-write and the formatting is pure and testable on
// its own.
func (h *Handler) render(r slog.Record) []byte {
	var b []byte
	b = append(b, r.Time.Format(timeFormat)...)
	b = append(b, ' ')
	b = append(b, r.Level.String()...)
	b = append(b, ' ')
	b = append(b, r.Message...)
	for _, pa := range h.attrs {
		b = appendAttr(b, pa.prefix, pa.attr)
	}
	r.Attrs(func(a slog.Attr) bool {
		b = appendAttr(b, h.prefix, a)
		return true
	})
	return append(b, '\n')
}

// WithAttrs returns a handler that renders attrs on every subsequent record.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	for _, a := range attrs {
		nh.attrs = append(nh.attrs, prefixedAttr{prefix: h.prefix, attr: a})
	}
	return &nh
}

// WithGroup returns a handler whose attribute keys are prefixed with the
// group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	nh := *h
	nh.prefix += name + "."
	return &nh
}

// appendAttr appends attr as key=value (with the group prefix) to b,
// descending into groups and rendering common value kinds; unknown kinds
// fall back to fmt's %v.
func appendAttr(b []byte, prefix string, a slog.Attr) []byte {
	if a.Equal(slog.Attr{}) {
		return b
	}
	a.Value = a.Value.Resolve()
	key := prefix + a.Key
	switch a.Value.Kind() {
	case slog.KindGroup:
		for _, ga := range a.Value.Group() {
			b = appendAttr(b, key+".", ga)
		}
		return b
	case slog.KindString:
		return appendKV(b, key, a.Value.String())
	case slog.KindBool:
		return appendKV(b, key, strconv.FormatBool(a.Value.Bool()))
	case slog.KindInt64:
		return appendKV(b, key, strconv.FormatInt(a.Value.Int64(), 10))
	case slog.KindUint64:
		return appendKV(b, key, strconv.FormatUint(a.Value.Uint64(), 10))
	case slog.KindFloat64:
		return appendKV(b, key, strconv.FormatFloat(a.Value.Float64(), 'g', -1, 64))
	case slog.KindDuration:
		return appendKV(b, key, a.Value.Duration().String())
	case slog.KindTime:
		return appendKV(b, key, a.Value.Time().Format(timeFormat))
	default:
		return appendKV(b, key, fmt.Sprint(a.Value.Any()))
	}
}

// appendKV appends a single space-separated key=value pair to b.
func appendKV(b []byte, key, val string) []byte {
	b = append(b, ' ')
	b = append(b, key...)
	b = append(b, '=')
	b = append(b, val...)
	return b
}
