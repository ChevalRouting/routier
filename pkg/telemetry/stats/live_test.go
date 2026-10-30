package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadIfaceCounters(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "eth0", "statistics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for file, value := range map[string]string{"rx_bytes": "1000\n", "tx_bytes": "2000\n", "rx_packets": "10\n", "tx_packets": "20\n"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "eth0", "operstate"), []byte("up\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := sysClassNet
	sysClassNet = root
	defer func() { sysClassNet = previous }()

	got, err := ReadIfaceCounters("eth0")
	if err != nil {
		t.Fatal(err)
	}

	if want := (IfaceCounters{RxBytes: 1000, TxBytes: 2000, RxPackets: 10, TxPackets: 20}); got != want {
		t.Fatalf("counters = %+v, want %+v", got, want)
	}

	if state := ReadOperState("eth0"); state != "up" {
		t.Fatalf("operstate = %q", state)
	}

	if _, err := ReadIfaceCounters("eth1"); err == nil {
		t.Fatal("missing interface accepted")
	}

	if state := ReadOperState("eth1"); state != "unknown" {
		t.Fatalf("missing operstate = %q", state)
	}
}

func TestNewLiveSample(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_500)
	prev := IfaceCounters{RxBytes: 1000, TxBytes: 5000, RxPackets: 10, TxPackets: 50}
	cur := IfaceCounters{RxBytes: 3000, TxBytes: 4000, RxPackets: 30, TxPackets: 60}

	got := NewLiveSample(prev, cur, 2*time.Second, at)
	if got.TS != at.UnixMilli() || got.RxBps != 1000 || got.RxPps != 10 || got.TxPps != 5 {
		t.Fatalf("sample = %+v", got)
	}

	if got.TxBps != 0 {
		t.Fatalf("counter reset rate = %v, want 0", got.TxBps)
	}

	if got.RxBytes != 3000 || got.TxBytes != 4000 {
		t.Fatalf("totals = %d/%d", got.RxBytes, got.TxBytes)
	}

	if zero := NewLiveSample(prev, cur, 0, at); zero.RxBps != 0 {
		t.Fatalf("zero elapsed rate = %v", zero.RxBps)
	}
}
