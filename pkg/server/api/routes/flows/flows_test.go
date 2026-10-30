package flows

import (
	"strings"
	"testing"
)

func TestPathDimensions(t *testing.T) {
	got, err := pathDimensions("")
	if err != nil || strings.Join(got, ",") != "src,service,dst" {
		t.Fatalf("default = %v, %v", got, err)
	}

	got, err = pathDimensions("in_if,out_if")
	if err != nil || len(got) != 2 {
		t.Fatalf("pair = %v, %v", got, err)
	}

	for _, raw := range []string{"src", "src,dst,in_if,out_if", "src,src", "src,conversation", "src,bogus"} {
		if _, err := pathDimensions(raw); err == nil {
			t.Fatalf("%q accepted", raw)
		}
	}
}
