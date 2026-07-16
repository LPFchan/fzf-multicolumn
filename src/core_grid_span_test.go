package fzf

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseGridSpanRecord(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		prefix     string
		grid       int
		wantText   string
		wantSpan   int
		wantErrSub string
	}{
		{"disabled compatibility", "@@5@@payload", "", 6, "@@5@@payload", 1, ""},
		{"valid", "@@5@@payload", "@@", 6, "payload", 5, ""},
		{"unicode payload", "§2§한글", "§", 6, "한글", 2, ""},
		{"ansi payload", "@@3@@\x1b[31mred", "@@", 6, "\x1b[31mred", 3, ""},
		{"malformed no digits", "@@@@payload", "@@", 6, "@@@@payload", 1, ""},
		{"malformed missing close", "@@5payload", "@@", 6, "@@5payload", 1, ""},
		{"zero", "@@0@@payload", "@@", 6, "", 0, "positive"},
		{"too wide", "@@7@@payload", "@@", 6, "", 0, "exceeds"},
		{"overflow", "@@999999999999999999999999999999@@payload", "@@", 6, "", 0, "overflows"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			text, span, err := parseGridSpanRecord([]byte(test.input), test.prefix, test.grid)
			if test.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrSub) {
					t.Fatalf("error = %v, want substring %q", err, test.wantErrSub)
				}
				return
			}
			if err != nil || !bytes.Equal(text, []byte(test.wantText)) || span != test.wantSpan {
				t.Fatalf("text=%q span=%d err=%v", text, span, err)
			}
		})
	}
}
