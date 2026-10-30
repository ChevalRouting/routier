package live

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBooted(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]bool{
		"tmpfs / tmpfs rw,relatime 0 0\n":                    true,
		"/dev/mapper/routier-root / xfs rw,noatime 0 0\n":    false,
		"proc /proc proc rw 0 0\noverlay / overlay rw 0 0\n": true,
	}
	for body, want := range cases {
		path := filepath.Join(dir, "mounts")
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}

		if got := Booted(path); got != want {
			t.Errorf("Booted(%q) = %v, want %v", body, got, want)
		}
	}

	if Booted(filepath.Join(dir, "absent")) {
		t.Error("missing mounts file reported a live boot")
	}
}

func TestParseDisks(t *testing.T) {
	data := []byte(`{"blockdevices":[
		{"name":"loop0","size":1000,"model":null,"rm":false,"type":"loop","mountpoint":"/.modloop"},
		{"name":"sda","size":"8589934592","model":"QEMU HARDDISK  ","rm":"0","type":"disk","mountpoint":null},
		{"name":"sr0","size":250000000,"model":"QEMU DVD-ROM","rm":true,"type":"rom","mountpoint":"/media/cdrom"},
		{"name":"sdb","size":16000000000,"model":"Flash","rm":true,"type":"disk","mountpoint":null,
			"children":[{"name":"sdb1","size":16000000000,"rm":true,"type":"part","mountpoint":"/media/sdb1"}]}
	]}`)

	disks, err := ParseDisks(data)
	if err != nil {
		t.Fatal(err)
	}

	want := []Disk{
		{Path: "/dev/sda", Size: 8589934592, Model: "QEMU HARDDISK"},
		{Path: "/dev/sdb", Size: 16000000000, Model: "Flash", Removable: true, BootMedia: true},
	}
	if !slices.Equal(disks, want) {
		t.Fatalf("disks = %#v, want %#v", disks, want)
	}
}

func TestTimezones(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"UTC", "Europe/Paris", "America/Argentina/Salta", "posix/Europe/Paris", "Etc/GMT+1", "zone.tab", "iso3166.tab"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := Timezones(root)
	want := []string{"America/Argentina/Salta", "Europe/Paris", "UTC"}
	if !slices.Equal(got, want) {
		t.Fatalf("Timezones = %v, want %v", got, want)
	}
}
