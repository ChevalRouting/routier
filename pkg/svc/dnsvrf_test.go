package svc

import "testing"

func TestDNSVRFNoConfiguration(t *testing.T) {
	if err := applyDNSVRFs(nil); err != nil {
		t.Fatal(err)
	}
}
