package bind

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestZonesQueriesLiveSerialsConcurrently(t *testing.T) {
	var calls atomic.Int32
	restore := SetRunner(func(string, ...string) ([]byte, error) {
		calls.Add(1)
		time.Sleep(40 * time.Millisecond)
		return []byte(";; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 1\n;; ANSWER SECTION:\na.example. 300 IN SOA ns.example. hostmaster.example. 42 1 1 1 1"), nil
	})
	defer restore()
	start := time.Now()
	got := New("").Zones([]ZoneInput{{Name: "a.example"}, {Name: "b.example"}, {Name: "c.example"}})
	if elapsed := time.Since(start); elapsed >= 100*time.Millisecond {
		t.Fatalf("zone queries ran serially: %v", elapsed)
	}

	if calls.Load() != 3 || len(got) != 3 {
		t.Fatalf("unexpected queries=%d zones=%d", calls.Load(), len(got))
	}
}
