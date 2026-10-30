package flows

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/telemetry/nflog"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	flushInterval   = time.Second
	expireInterval  = 5 * time.Second
	counterInterval = 20 * time.Second
	namesInterval   = 30 * time.Second
	pruneInterval   = time.Minute
	storeTimeout    = 10 * time.Second
)

type Store interface {
	InsertFlows(ctx context.Context, flows []types.Flow) error
	PruneFlows(ctx context.Context, before int64, keep int) error
	SaveFlowName(ctx context.Context, address, name string, resolvedAt int64) error
}

type agent struct {
	settings Settings
	db       Store
	exporter *exporter
	cache    *cache
	names    interfaceNames
	ptr      *nameResolver
	samples  uint64
}

type schedule struct {
	flush, expire, counters, names, prune time.Time
}

func dialCollectors(s Settings) ([]*collector, error) {
	var collectors []*collector
	var failures []string
	for _, addr := range s.Collectors {
		c, err := dialCollector(addr, s.Source)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", addr, err))
			continue
		}

		collectors = append(collectors, c)
	}

	if len(failures) > 0 {
		return collectors, errors.New("collectors unreachable: " + strings.Join(failures, "; "))
	}

	return collectors, nil
}

func Run(ctx context.Context, s Settings, store Store) (err error) {
	conn, err := nflog.Open(config.FlowLogGroup, config.FlowSnaplen)
	if err != nil {
		return err
	}

	defer func() { _ = conn.Close() }()

	collectors, dialErr := dialCollectors(s)
	now := time.Now()
	a := &agent{settings: s, db: store, exporter: newExporter(collectors, now), cache: newCache(s.Sampling), names: loadInterfaceNames(), ptr: newNameResolver(store, net.DefaultResolver.LookupAddr)}
	defer a.exporter.close()

	a.ptr.start(ctx)

	if dialErr != nil {
		log.Warn().Err(dialErr).Msg("flows: sflow export")
	}

	status.start(s, collectors, dialErr)

	defer func() { status.stop(err) }()

	log.Info().Int("sampling", s.Sampling).Strs("interfaces", s.Devices).Int("collectors", len(collectors)).Msg("flows: agent started")

	next := schedule{flush: now, expire: now, counters: now, names: now.Add(namesInterval), prune: now}
	for ctx.Err() == nil {
		overrun, err := conn.Read(a.handle)
		if err != nil {
			return fmt.Errorf("read nflog: %w", err)
		}

		var overruns uint64
		if overrun {
			overruns = 1
		}

		status.count(a.samples, overruns)
		a.samples = 0
		a.tick(ctx, &next, time.Now())
	}

	a.exporter.flush(time.Now())
	a.store(context.WithoutCancel(ctx), a.cache.expire(time.Now(), true))
	return nil
}

func (a *agent) handle(p nflog.Packet) {
	sampled, ok := ParsePacket(p.Payload, p.InDev, p.OutDev)
	if !ok {
		return
	}

	now := time.Now()
	a.samples++
	a.exporter.addFlow(sampled, ethernetFrame(p.HwHeader, sampled.Key.Family, p.Payload), a.settings.Sampling, now)
	if full := a.cache.add(sampled, now); full {
		a.store(context.Background(), a.cache.expire(now, true))
	}
}

func (a *agent) tick(ctx context.Context, next *schedule, now time.Time) {
	if !now.Before(next.flush) {
		a.exporter.flush(now)
		next.flush = now.Add(flushInterval)
	}

	if !now.Before(next.counters) {
		a.sendCounters(now)
		next.counters = now.Add(counterInterval)
	}

	if !now.Before(next.names) {
		a.names = loadInterfaceNames()
		next.names = now.Add(namesInterval)
	}

	if !now.Before(next.expire) {
		a.store(ctx, a.cache.expire(now, false))
		next.expire = now.Add(expireInterval)
	}

	if !now.Before(next.prune) {
		a.prune(ctx, now)
		next.prune = now.Add(pruneInterval)
	}
}

func (a *agent) sendCounters(now time.Time) {
	for _, device := range a.settings.Devices {
		if c, ok := interfaceCounters(device); ok {
			a.exporter.addCounters(c, now)
		}
	}
}

func (a *agent) toFlow(r Record) types.Flow {
	k := r.Key
	return types.Flow{
		Start: r.First.Unix(), End: r.Last.Unix(),
		InIf: a.names.name(k.InDev), OutIf: a.names.name(k.OutDev),
		Family: int(k.Family), Proto: int(k.Proto),
		Src: k.Src.String(), Dst: k.Dst.String(), SrcPort: int(k.SrcPort), DstPort: int(k.DstPort),
		Packets: int64(r.Packets), Bytes: int64(r.Bytes), Samples: int64(r.Samples),
	}
}

func (a *agent) store(ctx context.Context, records []Record) {
	if len(records) == 0 {
		return
	}

	now := time.Now()
	flows := make([]types.Flow, 0, len(records))
	for _, r := range records {
		flows = append(flows, a.toFlow(r))
		a.ptr.want(now, r.Key.Src, r.Key.Dst)
	}

	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	if err := a.db.InsertFlows(ctx, flows); err != nil {
		log.Warn().Err(err).Int("flows", len(flows)).Msg("flows: store")
	}
}

func (a *agent) prune(ctx context.Context, now time.Time) {
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	if err := a.db.PruneFlows(ctx, now.Unix()-a.settings.Retention, a.settings.MaxFlows); err != nil {
		log.Warn().Err(err).Msg("flows: prune")
	}
}
