package probe

import "testing"

func TestParseAverage(t *testing.T) {
	for _, output := range []string{
		"rtt min/avg/max/mdev = 10.100/12.345/15.900/1.2 ms",
		"round-trip min/avg/max = 1.000/2.500/4.000 ms",
	} {
		got, ok := ParseAverage(output)
		if !ok || got <= 0 {
			t.Fatalf("failed to parse %q: %v %v", output, got, ok)
		}
	}
}
