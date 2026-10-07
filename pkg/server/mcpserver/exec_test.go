package mcpserver

import "testing"

func TestParseExecOutputExtractsBetweenMarkers(t *testing.T) {
	marker := "RTRdeadbeef"
	start := marker + "START"
	end := marker + "END"

	raw := "\x1b[?2004hrouter:~# printf '%sSTART\\n' '" + marker + "'; echo hello; printf '%sEND %d\\n' '" + marker + "' \"$?\"; exit\r\n" +
		"\x1b[?2004l\r" + start + "\r\n" +
		"hello\r\n" +
		end + " 0\r\n"

	res, err := parseExecOutput("echo hello", raw, start, end)
	if err != nil {
		t.Fatal(err)
	}

	if res.Output != "hello" {
		t.Fatalf("output = %q, want %q", res.Output, "hello")
	}

	if res.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", res.ExitCode)
	}
}

func TestParseExecOutputCapturesNonZeroExit(t *testing.T) {
	marker := "RTRcafe"
	start := marker + "START"
	end := marker + "END"

	raw := start + "\r\n" + "boom\r\n" + "line two\r\n" + end + " 3\r\n"

	res, err := parseExecOutput("false", raw, start, end)
	if err != nil {
		t.Fatal(err)
	}

	if res.Output != "boom\nline two" {
		t.Fatalf("output = %q, want %q", res.Output, "boom\nline two")
	}

	if res.ExitCode != 3 {
		t.Fatalf("exit code = %d, want 3", res.ExitCode)
	}
}

func TestParseExecOutputMissingStartMarker(t *testing.T) {
	if _, err := parseExecOutput("cmd", "some noise without markers", "RTRxSTART", "RTRxEND"); err == nil {
		t.Fatal("expected error when start marker is absent")
	}
}
