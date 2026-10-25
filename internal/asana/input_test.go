package asana

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

func TestParseDueDate(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"  ", "", false},
		{"today", today, false},
		{"Tomorrow", tomorrow, false},
		{"2026-11-01", "2026-11-01", false},
		{"2026-13-01", "", true},
		{"11/01/2026", "", true},
		{"next week", "", true},
	}

	for _, tt := range tests {
		got, err := parseDueDate(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDueDate(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDueDate(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestPromptLineTrimsInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  first answer  \nlast answer without newline"))

	first, err := promptLine(reader, "")
	if err != nil || first != "first answer" {
		t.Errorf("first = %q, %v; want %q", first, err, "first answer")
	}

	last, err := promptLine(reader, "")
	if err != nil || last != "last answer without newline" {
		t.Errorf("last = %q, %v; want %q", last, err, "last answer without newline")
	}

	empty, err := promptLine(reader, "")
	if err != nil || empty != "" {
		t.Errorf("after EOF = %q, %v; want empty", empty, err)
	}
}
