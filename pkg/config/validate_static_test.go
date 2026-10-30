package config

import (
	"fmt"
	"strings"
	"testing"
)

func staticErrors(routes ...StaticRoute) string {
	var messages []string
	add := func(msg string, args ...any) { messages = append(messages, fmt.Sprintf(msg, args...)) }
	validateStaticRoutes(routes, "routing.static", add)
	return strings.Join(messages, "\n")
}

func TestStaticRouteSource(t *testing.T) {
	good := []StaticRoute{
		{Destination: "::/0", Via: "fe80::1", Dev: "wan", Src: "2001:db8::10"},
		{Destination: "0.0.0.0/0", Via: "192.0.2.1", Src: "192.0.2.10"},
	}

	if got := staticErrors(good...); got != "" {
		t.Fatalf("valid sources were rejected:\n%s", got)
	}

	cases := map[string]StaticRoute{
		"is not a valid IP":                 {Destination: "::/0", Dev: "wan", Src: "2001:db8::/64"},
		"is not in the same address family": {Destination: "::/0", Dev: "wan", Src: "192.0.2.10"},
	}

	for want, route := range cases {
		if got := staticErrors(route); !strings.Contains(got, want) {
			t.Errorf("src %q: got %q, want %q", route.Src, got, want)
		}
	}
}
