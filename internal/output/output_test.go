package output

import (
	"bytes"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		pretty   bool
		expected string
	}{
		{
			name:     "compact object",
			data:     map[string]string{"key": "value"},
			pretty:   false,
			expected: `{"key":"value"}`,
		},
		{
			name:     "pretty object",
			data:     map[string]string{"key": "value"},
			pretty:   true,
			expected: "{\n  \"key\": \"value\"\n}",
		},
		{
			name:     "nil data",
			data:     nil,
			pretty:   false,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Print(&buf, tt.data, tt.pretty)
			if buf.String() != tt.expected+"\n" {
				t.Errorf("Print() = %q, want %q", buf.String(), tt.expected+"\n")
			}
		})
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	PrintError(&buf, "test error")
	expected := `{"error":"test error"}` + "\n"
	if buf.String() != expected {
		t.Errorf("PrintError() = %q, want %q", buf.String(), expected)
	}
}
