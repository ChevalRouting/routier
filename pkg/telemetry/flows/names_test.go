package flows

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/types"
)

type nameStore struct {
	mu    sync.Mutex
	names map[string]string
}

func (s *nameStore) InsertFlows(context.Context, []types.Flow) error { return nil }

func (s *nameStore) PruneFlows(context.Context, int64, int) error { return nil }

func (s *nameStore) SaveFlowName(_ context.Context, address, name string, _ int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.names[address] = name
	return nil
}

type fakeLookup struct {
	mu    sync.Mutex
	calls map[string]int
}

func (f *fakeLookup) lookup(_ context.Context, address string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[address]++
	switch address {
	case "192.0.2.1":
		return []string{"host.example."}, nil
	case "192.0.2.3":
		return nil, &net.DNSError{Err: "timeout", IsTimeout: true}
	}

	return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
}

func drain(n *nameResolver) {
	for {
		select {
		case a := <-n.queue:
			n.resolve(context.Background(), a)
		default:
			return
		}
	}
}

func TestNameResolverCachesLookups(t *testing.T) {
	store := &nameStore{names: map[string]string{}}
	dns := &fakeLookup{calls: map[string]int{}}
	n := newNameResolver(store, dns.lookup)
	now := time.Now()
	addrs := []netip.Addr{
		netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2"), netip.MustParseAddr("192.0.2.3"),
		netip.MustParseAddr("ff02::1"), netip.MustParseAddr("fe80::1"), netip.MustParseAddr("0.0.0.0"),
	}

	n.want(now, addrs...)
	drain(n)
	n.want(now, addrs...)
	drain(n)

	if store.names["192.0.2.1"] != "host.example" || store.names["192.0.2.2"] != "" || len(store.names) != 2 {
		t.Fatalf("stored names = %+v", store.names)
	}

	if len(dns.calls) != 3 || dns.calls["192.0.2.1"] != 1 || dns.calls["192.0.2.2"] != 1 || dns.calls["192.0.2.3"] != 1 {
		t.Fatalf("lookups = %+v", dns.calls)
	}

	n.want(now.Add(missingNameTTL+time.Second), addrs...)
	drain(n)

	if dns.calls["192.0.2.1"] != 1 || dns.calls["192.0.2.2"] != 2 || dns.calls["192.0.2.3"] != 2 {
		t.Fatalf("lookups after negative expiry = %+v", dns.calls)
	}
}
