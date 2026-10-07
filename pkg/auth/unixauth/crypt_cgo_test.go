//go:build linux && cgo

package unixauth

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCryptVerify(t *testing.T) {
	out, err := exec.Command("openssl", "passwd", "-6", "-salt", "abcd1234", "s3cret").Output()
	if err != nil {
		t.Skipf("openssl unavailable: %v", err)
	}

	hash := strings.TrimSpace(string(out))
	if !strings.HasPrefix(hash, "$6$") {
		t.Fatalf("unexpected hash: %q", hash)
	}

	ok, err := cryptVerify("s3cret", hash)
	if err != nil {
		t.Fatalf("verify correct password: %v", err)
	}

	if !ok {
		t.Fatalf("correct password rejected for %q", hash)
	}

	ok, err = cryptVerify("wrong", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}

	if ok {
		t.Fatalf("wrong password accepted for %q", hash)
	}
}
