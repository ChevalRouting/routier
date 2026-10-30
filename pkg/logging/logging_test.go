package logging

import "testing"

func TestIndentErrorLines(t *testing.T) {
	cases := map[string]string{
		`"single"`:        "single",
		"plain":           "plain",
		`"first\nsecond"`: "\n  first\n  second",
	}

	for in, want := range cases {
		if got := indentErrorLines(in); got != want {
			t.Errorf("indentErrorLines(%q) = %q, want %q", in, got, want)
		}
	}
}
