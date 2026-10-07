package bind

import "testing"

func TestParseQueryLog(t *testing.T) {
	cases := []parseQueryLogCase{
		{
			name: "ipv4 with edns and dest",
			line: "07-Oct-2026 12:34:56.789 queries: client @0x7f0a1c000d68 192.168.1.10#54321 (example.com): query: example.com IN A +E(0)K (192.168.1.1)",
			want: LogQuery{Time: "07-Oct-2026 12:34:56.789", Client: "192.168.1.10", Port: "54321", Name: "example.com", Class: "IN", Type: "A", Flags: "+E(0)K", Dest: "192.168.1.1"},
		},
		{
			name: "with severity, no dest",
			line: "07-Oct-2026 12:35:00.001 queries: info: client 192.168.1.20#5000 (foo.lan): query: foo.lan IN AAAA +",
			want: LogQuery{Time: "07-Oct-2026 12:35:00.001", Client: "192.168.1.20", Port: "5000", Name: "foo.lan", Class: "IN", Type: "AAAA", Flags: "+"},
		},
		{
			name: "ipv6 client and dest",
			line: "07-Oct-2026 12:35:01.500 queries: client 2001:db8::1#5353 (a.b.): query: a.b. IN HTTPS +E(0)KV (2001:db8::53)",
			want: LogQuery{Time: "07-Oct-2026 12:35:01.500", Client: "2001:db8::1", Port: "5353", Name: "a.b", Class: "IN", Type: "HTTPS", Flags: "+E(0)KV", Dest: "2001:db8::53"},
		},
	}

	for _, c := range cases {
		got, ok := ParseQueryLog(c.line)
		if !ok {
			t.Fatalf("%s: ParseQueryLog returned false", c.name)
		}

		got.Raw = ""
		if got != c.want {
			t.Errorf("%s:\n got  %+v\n want %+v", c.name, got, c.want)
		}
	}
}

func TestParseQueryLogRejectsNonQuery(t *testing.T) {
	for _, line := range []string{
		"07-Oct-2026 12:34:56.789 general: info: zone example.com/IN: loaded serial 3",
		"==> /var/log/named/named.log <==",
		"",
	} {
		if _, ok := ParseQueryLog(line); ok {
			t.Errorf("expected non-query line to be rejected: %q", line)
		}
	}
}

type parseQueryLogCase struct {
	name string
	line string
	want LogQuery
}
