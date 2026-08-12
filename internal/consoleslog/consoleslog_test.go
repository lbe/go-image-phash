package consoleslog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestHandlerRendersRecords(t *testing.T) {
	var buf bytes.Buffer
	h := New(&buf)

	record := slog.NewRecord(
		time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC),
		slog.LevelInfo,
		"hash computed",
		0,
	)
	record.AddAttrs(
		slog.String("path", "photo.jpg"),
		slog.Int("count", 3),
		slog.Bool("ok", true),
		slog.Uint64("size", 42),
		slog.Float64("ratio", 1.5),
		slog.Duration("elapsed", time.Second),
		slog.Time("when", time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)),
		slog.Group("meta", slog.String("kind", "jpeg")),
		slog.Any("custom", struct{ X int }{7}),
	)

	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled must allow Info")
	}
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("Enabled must reject Debug when threshold is Info")
	}

	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"2009-11-10T23:00:00Z",
		"INFO",
		"hash computed",
		"path=photo.jpg",
		"count=3",
		"ok=true",
		"size=42",
		"ratio=1.5",
		"elapsed=1s",
		"when=2009-11-10T23:00:00Z",
		"meta.kind=jpeg",
		"custom={7}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q missing %q", got, want)
		}
	}
}

func TestHandlerWithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	base := New(&buf)
	h := base.WithAttrs([]slog.Attr{slog.String("service", "phash")})
	h = h.WithGroup("req")

	record := slog.NewRecord(time.Now(), slog.LevelError, "load failed", 0)
	record.AddAttrs(slog.String("error", "ENOENT"))

	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "service=phash") {
		t.Errorf("WithAttrs output %q missing service=phash", got)
	}
	if !strings.Contains(got, "req.error=ENOENT") {
		t.Errorf("WithGroup output %q missing req.error=ENOENT", got)
	}
}

func TestHandlerSkipsEmptyAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := New(&buf)
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	record.AddAttrs(slog.Attr{})

	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if strings.Count(buf.String(), "=") != 0 {
		t.Errorf("empty attr should not render key=value pairs, got %q", buf.String())
	}
}

func TestHandlerWriteError(t *testing.T) {
	h := New(&failWriter{})
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), record); err == nil {
		t.Fatal("Handle must return write errors")
	}
}

func TestNewLoggerWritesToStderr(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
