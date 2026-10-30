package flows

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	nameTTL        = time.Hour
	missingNameTTL = 10 * time.Minute
	lookupTimeout  = 2 * time.Second
	lookupWorkers  = 4
	lookupQueue    = 1024
	maxKnownNames  = 65536
)

var broadcast = netip.AddrFrom4([4]byte{255, 255, 255, 255})

type lookupFunc func(ctx context.Context, address string) ([]string, error)

type nameResolver struct {
	mu     sync.Mutex
	expiry map[netip.Addr]time.Time
	queue  chan netip.Addr
	lookup lookupFunc
	store  Store
}

func newNameResolver(store Store, lookup lookupFunc) *nameResolver {
	return &nameResolver{expiry: map[netip.Addr]time.Time{}, queue: make(chan netip.Addr, lookupQueue), lookup: lookup, store: store}
}

func resolvable(a netip.Addr) bool {
	return a.IsValid() && !a.IsUnspecified() && !a.IsLoopback() && !a.IsMulticast() && !a.IsLinkLocalUnicast() && a != broadcast
}

func (n *nameResolver) start(ctx context.Context) {
	for range lookupWorkers {
		go n.work(ctx)
	}
}

func (n *nameResolver) want(now time.Time, addrs ...netip.Addr) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if len(n.expiry) > maxKnownNames {
		n.forgetExpired(now)
	}

	for _, a := range addrs {
		if !resolvable(a) || n.expiry[a].After(now) {
			continue
		}

		select {
		case n.queue <- a:
			n.expiry[a] = now.Add(missingNameTTL)
		default:
			return
		}
	}
}

func (n *nameResolver) forgetExpired(now time.Time) {
	for a, until := range n.expiry {
		if !until.After(now) {
			delete(n.expiry, a)
		}
	}
}

func (n *nameResolver) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case a := <-n.queue:
			n.resolve(ctx, a)
		}
	}
}

func notFound(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

func (n *nameResolver) resolve(ctx context.Context, a netip.Addr) {
	lookupCtx, cancel := context.WithTimeout(ctx, lookupTimeout)
	names, err := n.lookup(lookupCtx, a.String())
	cancel()

	if err != nil && !notFound(err) {
		return
	}

	name, ttl := "", missingNameTTL
	if len(names) > 0 {
		name, ttl = strings.TrimSuffix(names[0], "."), nameTTL
	}

	now := time.Now()
	storeCtx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	if err := n.store.SaveFlowName(storeCtx, a.String(), name, now.Unix()); err != nil {
		log.Warn().Err(err).Str("address", a.String()).Msg("flows: store name")
		return
	}

	n.mu.Lock()
	n.expiry[a] = now.Add(ttl)
	n.mu.Unlock()
}
