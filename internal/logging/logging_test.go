package logging

import (
	"context"
	"regexp"
	"testing"
)

func TestNewRequestID_Format(t *testing.T) {
	id := NewRequestID()
	// UUID v4 格式: 8-4-4-4-12 hex
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
	if !regexp.MustCompile(pattern).MatchString(id) {
		t.Errorf("NewRequestID() = %q, not valid UUID v4", id)
	}
}

func TestNewRequestID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewRequestID()
		if seen[id] {
			t.Fatalf("duplicate requestId: %s", id)
		}
		seen[id] = true
	}
}

func TestWithRequestID_RoundTrip(t *testing.T) {
	ctx := context.Background()
	id := "test-request-id-123"
	ctx = WithRequestID(ctx, id)

	got := RequestIDFrom(ctx)
	if got != id {
		t.Errorf("RequestIDFrom() = %q, want %q", got, id)
	}
}

func TestRequestIDFrom_Empty(t *testing.T) {
	ctx := context.Background()
	got := RequestIDFrom(ctx)
	if got != "" {
		t.Errorf("RequestIDFrom() = %q, want empty", got)
	}
}
