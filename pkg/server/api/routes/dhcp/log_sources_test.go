package dhcp

import (
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/render"
)

func TestLeaseLogFiles(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []string
	}{
		{"", []string{render.KeaLog4, render.KeaLog6}},
		{"kea-dhcp4", []string{render.KeaLog4}},
		{"kea-dhcp6", []string{render.KeaLog6}},
		{"kea-dhcp-ddns", []string{render.KeaLogDDNS}},
		{"/etc/passwd", nil},
		{"unknown", nil},
	} {
		t.Run(tc.source, func(t *testing.T) {
			got, ok := leaseLogFiles(tc.source)
			if ok != (tc.want != nil) || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("leaseLogFiles(%q) = %v, %v; want %v", tc.source, got, ok, tc.want)
			}
		})
	}
}
