package artifacterr

import "testing"

func TestParseNft(t *testing.T) {
	out := "/tmp/routier-nftcheck-123.nft:42:15-20: Error: Could not resolve hostname: foo\n"
	errs := Parse(ToolNft, "/etc/nftables.d/routier.nft", out)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %+v", len(errs), errs)
	}

	e := errs[0]
	if e.Line != 42 || e.Column != 15 {
		t.Fatalf("want line 42 col 15, got line %d col %d", e.Line, e.Column)
	}

	if e.Message != "Could not resolve hostname: foo" {
		t.Fatalf("unexpected message: %q", e.Message)
	}

	if e.Tool != ToolNft || e.Dest != "/etc/nftables.d/routier.nft" {
		t.Fatalf("tool/dest not stamped: %+v", e)
	}
}

func TestParseVtysh(t *testing.T) {
	out := "line 12: % Invalid input detected at '^' marker.\n% Unknown command\n"
	errs := Parse(ToolVtysh, "/etc/frr/frr.conf", out)
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %d: %+v", len(errs), errs)
	}

	if errs[0].Line != 12 || errs[0].Message != "% Invalid input detected at '^' marker." {
		t.Fatalf("unexpected first error: %+v", errs[0])
	}

	if errs[1].Line != 0 || errs[1].Message != "Unknown command" {
		t.Fatalf("unexpected second error: %+v", errs[1])
	}
}

func TestParseRadvd(t *testing.T) {
	out := "radvd: /etc/radvd.conf:12 error: unknown drtr preference\n"
	errs := Parse(ToolRadvd, "/etc/radvd.conf", out)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Line != 12 {
		t.Fatalf("want line 12, got %d", errs[0].Line)
	}
}

func TestParseKeaParen(t *testing.T) {
	out := "ERROR Error: unable to configure server (/etc/kea/kea-dhcp4.conf:12:34)\n"
	errs := Parse(ToolKea, "/etc/kea/kea-dhcp4.conf", out)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %+v", len(errs), errs)
	}

	e := errs[0]
	if e.Line != 12 || e.Column != 34 {
		t.Fatalf("want line 12 col 34, got line %d col %d", e.Line, e.Column)
	}

	if e.Message != "ERROR Error: unable to configure server" {
		t.Fatalf("unexpected message: %q", e.Message)
	}
}

func TestParseKeaLead(t *testing.T) {
	out := "/etc/kea/kea-dhcp4.conf:5:7: unexpected end of file\n"
	errs := Parse(ToolKea, "/etc/kea/kea-dhcp4.conf", out)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Line != 5 || errs[0].Column != 7 || errs[0].Message != "unexpected end of file" {
		t.Fatalf("unexpected error: %+v", errs[0])
	}
}

func TestParseFallback(t *testing.T) {
	out := "some unstructured failure with no location\n"
	errs := Parse(ToolNft, "/etc/nftables.d/routier.nft", out)
	if len(errs) != 1 || errs[0].Line != 0 {
		t.Fatalf("want 1 line-0 error, got %+v", errs)
	}

	if errs[0].Message != "some unstructured failure with no location" {
		t.Fatalf("unexpected message: %q", errs[0].Message)
	}
}

func TestParseEmpty(t *testing.T) {
	if errs := Parse(ToolNft, "x", "   \n"); errs != nil {
		t.Fatalf("want nil for empty output, got %+v", errs)
	}
}
