package ws

import "testing"

func TestConsoleUserRefusesSessionsWithoutUnixUser(t *testing.T) {
	for _, username := range []string{"", "root", "no-such-routier-user"} {
		if _, err := consoleUser(username); err == nil {
			t.Errorf("consoleUser(%q) should refuse the session", username)
		}
	}
}
