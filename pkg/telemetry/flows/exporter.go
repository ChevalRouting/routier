package flows

import (
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/telemetry/sflow"
)

const maxDatagram = 1400

type collector struct {
	addr  netip.AddrPort
	conn  *net.UDPConn
	agent net.IP

	mu      sync.Mutex
	sent    uint64
	errors  uint64
	lastErr string
}

func dialCollector(addr netip.AddrPort, source netip.Addr) (*collector, error) {
	var local *net.UDPAddr
	if source.IsValid() {
		local = &net.UDPAddr{IP: source.AsSlice()}
	}

	conn, err := net.DialUDP("udp", local, net.UDPAddrFromAddrPort(addr))
	if err != nil {
		return nil, err
	}

	agent := conn.LocalAddr().(*net.UDPAddr).IP
	return &collector{addr: addr, conn: conn, agent: agent}, nil
}

func (c *collector) send(d sflow.Datagram) {
	d.Agent = c.agent
	_, err := c.conn.Write(d.Marshal())

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.errors++
		c.lastErr = err.Error()
		return
	}

	c.sent++
	c.lastErr = ""
}

type exporter struct {
	collectors []*collector
	started    time.Time
	sequence   uint32
	pending    []sflow.Sample
	size       int
	flowSeq    map[uint32]uint32
	counterSeq map[uint32]uint32
	pool       map[uint32]uint32
}

func newExporter(collectors []*collector, started time.Time) *exporter {
	return &exporter{
		collectors: collectors,
		started:    started,
		size:       sflow.HeaderLen(net.IPv6zero),
		flowSeq:    map[uint32]uint32{},
		counterSeq: map[uint32]uint32{},
		pool:       map[uint32]uint32{},
	}
}

func (e *exporter) add(s sflow.Sample, now time.Time) {
	if len(e.collectors) == 0 {
		return
	}

	length := sflow.EncodedLen(s)
	if e.size+length > maxDatagram {
		e.flush(now)
	}

	e.pending = append(e.pending, s)
	e.size += length
}

func outputIndex(outDev uint32) uint32 {
	if outDev == 0 {
		return sflow.InterfaceLocal
	}

	return outDev
}

func (e *exporter) addFlow(s Sampled, frame []byte, rate int, now time.Time) {
	source := s.Key.InDev
	e.flowSeq[source]++
	e.pool[source] += uint32(rate)
	e.add(sflow.FlowSample{
		Sequence:     e.flowSeq[source],
		SourceIndex:  source,
		SamplingRate: uint32(rate),
		SamplePool:   e.pool[source],
		Input:        source,
		Output:       outputIndex(s.Key.OutDev),
		Header: sflow.RawHeader{
			Protocol:    sflow.HeaderEthernet,
			FrameLength: s.Length + ethernetHeaderLen + ethernetFCSLen,
			Stripped:    ethernetFCSLen,
			Data:        frame,
		},
	}, now)
}

func (e *exporter) addCounters(c sflow.InterfaceCounters, now time.Time) {
	e.counterSeq[c.Index]++
	e.add(sflow.CounterSample{Sequence: e.counterSeq[c.Index], SourceIndex: c.Index, Counters: c}, now)
}

func (e *exporter) flush(now time.Time) {
	if len(e.pending) == 0 {
		return
	}

	e.sequence++
	d := sflow.Datagram{Sequence: e.sequence, Uptime: uint32(now.Sub(e.started).Milliseconds()), Samples: e.pending}
	for _, c := range e.collectors {
		c.send(d)
	}

	e.pending = nil
	e.size = sflow.HeaderLen(net.IPv6zero)
}

func (e *exporter) close() {
	for _, c := range e.collectors {
		_ = c.conn.Close()
	}
}
