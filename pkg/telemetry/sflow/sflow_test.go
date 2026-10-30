package sflow

import (
	"encoding/binary"
	"net"
	"testing"
)

type reader struct {
	t   *testing.T
	buf []byte
}

func (r *reader) u32() uint32 {
	r.t.Helper()
	if len(r.buf) < 4 {
		r.t.Fatal("datagram is too short")
	}

	v := binary.BigEndian.Uint32(r.buf)
	r.buf = r.buf[4:]
	return v
}

func (r *reader) bytes(n int) []byte {
	r.t.Helper()
	if len(r.buf) < n {
		r.t.Fatal("datagram is too short")
	}

	v := r.buf[:n]
	r.buf = r.buf[n:]
	return v
}

func TestMarshalFlowSample(t *testing.T) {
	header := []byte{0x45, 0, 0, 40, 1}
	d := Datagram{
		Agent:    net.ParseIP("192.0.2.1"),
		Sequence: 7,
		Uptime:   1000,
		Samples: []Sample{FlowSample{
			Sequence: 3, SourceIndex: 2, SamplingRate: 1000, SamplePool: 3000,
			Input: 2, Output: InterfaceLocal,
			Header: RawHeader{Protocol: HeaderIPv4, FrameLength: 40, Data: header},
		}},
	}

	b := d.Marshal()
	if len(b) != HeaderLen(d.Agent)+EncodedLen(d.Samples[0]) {
		t.Fatalf("length %d does not match the computed sizes", len(b))
	}

	r := &reader{t: t, buf: b}
	for _, want := range []uint32{Version, addressIPv4} {
		if got := r.u32(); got != want {
			t.Fatalf("header field = %d, want %d", got, want)
		}
	}

	if agent := net.IP(r.bytes(4)); !agent.Equal(d.Agent) {
		t.Fatalf("agent = %s", agent)
	}

	for _, want := range []uint32{0, 7, 1000, 1, formatFlowSample} {
		if got := r.u32(); got != want {
			t.Fatalf("datagram field = %d, want %d", got, want)
		}
	}

	if length := r.u32(); int(length) != len(r.buf) {
		t.Fatalf("sample length %d, remaining %d", length, len(r.buf))
	}

	for _, want := range []uint32{3, 2, 1000, 3000, 0, 2, InterfaceLocal, 1, formatRawHeader, 24, HeaderIPv4, 40, 0, 5} {
		if got := r.u32(); got != want {
			t.Fatalf("flow sample field = %#x, want %#x", got, want)
		}
	}

	if got := r.bytes(8); string(got[:5]) != string(header) || got[5] != 0 || got[7] != 0 {
		t.Fatalf("header bytes = %x", got)
	}

	if len(r.buf) != 0 {
		t.Fatalf("%d trailing bytes", len(r.buf))
	}
}

func TestMarshalCounterSampleIPv6Agent(t *testing.T) {
	d := Datagram{
		Agent:   net.ParseIP("2001:db8::1"),
		Samples: []Sample{CounterSample{Sequence: 1, SourceIndex: 4, Counters: InterfaceCounters{Index: 4, Speed: 1e9, InOctets: 1 << 40}}},
	}

	b := d.Marshal()
	r := &reader{t: t, buf: b}
	_ = r.u32()
	if kind := r.u32(); kind != addressIPv6 {
		t.Fatalf("address type = %d", kind)
	}

	_ = r.bytes(16)
	for range 4 {
		_ = r.u32()
	}

	if format := r.u32(); format != formatCounterSample {
		t.Fatalf("sample format = %d", format)
	}

	_ = r.u32()
	for _, want := range []uint32{1, 4, 1, formatIfCounters, ifCountersLength, 4} {
		if got := r.u32(); got != want {
			t.Fatalf("counter field = %d, want %d", got, want)
		}
	}

	if len(r.buf) != ifCountersLength-4 {
		t.Fatalf("counter body has %d bytes left, want %d", len(r.buf), ifCountersLength-4)
	}
}
