package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestMultiHandler_WritesToAll(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, nil)
	h2 := slog.NewTextHandler(&buf2, nil)

	logger := slog.New(newMultiHandler(h1, h2))
	logger.Info("test message", "key", "value")

	if buf1.Len() == 0 {
		t.Error("handler 1 received no output")
	}
	if buf2.Len() == 0 {
		t.Error("handler 2 received no output")
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	// h1 allows only ERROR, h2 allows INFO+
	h1 := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	h2 := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	mh := newMultiHandler(h1, h2)

	// Should be enabled for INFO because h2 allows it
	if !mh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("multiHandler should be enabled for INFO when at least one handler allows it")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, nil)
	h2 := slog.NewTextHandler(&buf2, nil)

	mh := newMultiHandler(h1, h2)
	mhWithAttrs := mh.WithAttrs([]slog.Attr{slog.String("service", "test")})

	logger := slog.New(mhWithAttrs)
	logger.Info("test")

	if !bytes.Contains(buf1.Bytes(), []byte("service=test")) {
		t.Errorf("handler 1 should contain service attr, got: %s", buf1.String())
	}
	if !bytes.Contains(buf2.Bytes(), []byte("service=test")) {
		t.Errorf("handler 2 should contain service attr, got: %s", buf2.String())
	}
}
