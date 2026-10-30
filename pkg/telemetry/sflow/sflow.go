package sflow

import (
	"encoding/binary"
	"net"
)

const (
	Version = 5

	HeaderEthernet = 1
	HeaderIPv4     = 11
	HeaderIPv6     = 12

	InterfaceLocal = 0x3fffffff

	addressIPv4 = 1
	addressIPv6 = 2

	formatFlowSample    = 1
	formatCounterSample = 2
	formatRawHeader     = 1
	formatIfCounters    = 1

	ifCountersLength = 88

	StatusAdminUp = 1
	StatusOperUp  = 2
)

type Sample interface {
	format() uint32
	appendBody(b []byte) []byte
}

type Datagram struct {
	Agent    net.IP
	SubAgent uint32
	Sequence uint32
	Uptime   uint32
	Samples  []Sample
}

type RawHeader struct {
	Protocol    uint32
	FrameLength uint32
	Stripped    uint32
	Data        []byte
}

type FlowSample struct {
	Sequence     uint32
	SourceIndex  uint32
	SamplingRate uint32
	SamplePool   uint32
	Drops        uint32
	Input        uint32
	Output       uint32
	Header       RawHeader
}

type InterfaceCounters struct {
	Index              uint32
	Type               uint32
	Speed              uint64
	Direction          uint32
	Status             uint32
	InOctets           uint64
	InUnicastPackets   uint32
	InMulticastPackets uint32
	InBroadcastPackets uint32
	InDiscards         uint32
	InErrors           uint32
	InUnknownProtos    uint32
	OutOctets          uint64
	OutUnicastPackets  uint32
	OutMulticast       uint32
	OutBroadcast       uint32
	OutDiscards        uint32
	OutErrors          uint32
	Promiscuous        uint32
}

type CounterSample struct {
	Sequence    uint32
	SourceIndex uint32
	Counters    InterfaceCounters
}

func be32(b []byte, v uint32) []byte {
	return binary.BigEndian.AppendUint32(b, v)
}

func be64(b []byte, v uint64) []byte {
	return binary.BigEndian.AppendUint64(b, v)
}

func padded(b, data []byte) []byte {
	b = append(b, data...)
	return append(b, make([]byte, (4-len(data)%4)%4)...)
}

func appendRecord(b []byte, format uint32, body []byte) []byte {
	b = be32(b, format)
	b = be32(b, uint32(len(body)))
	return append(b, body...)
}

func (FlowSample) format() uint32 { return formatFlowSample }

func (s FlowSample) appendBody(b []byte) []byte {
	b = be32(b, s.Sequence)
	b = be32(b, s.SourceIndex)
	b = be32(b, s.SamplingRate)
	b = be32(b, s.SamplePool)
	b = be32(b, s.Drops)
	b = be32(b, s.Input)
	b = be32(b, s.Output)
	b = be32(b, 1)

	header := be32(nil, s.Header.Protocol)
	header = be32(header, s.Header.FrameLength)
	header = be32(header, s.Header.Stripped)
	header = be32(header, uint32(len(s.Header.Data)))
	header = padded(header, s.Header.Data)
	return appendRecord(b, formatRawHeader, header)
}

func (CounterSample) format() uint32 { return formatCounterSample }

func (s CounterSample) appendBody(b []byte) []byte {
	b = be32(b, s.Sequence)
	b = be32(b, s.SourceIndex)
	b = be32(b, 1)

	c := s.Counters
	body := make([]byte, 0, ifCountersLength)
	body = be32(body, c.Index)
	body = be32(body, c.Type)
	body = be64(body, c.Speed)
	body = be32(body, c.Direction)
	body = be32(body, c.Status)
	body = be64(body, c.InOctets)
	body = be32(body, c.InUnicastPackets)
	body = be32(body, c.InMulticastPackets)
	body = be32(body, c.InBroadcastPackets)
	body = be32(body, c.InDiscards)
	body = be32(body, c.InErrors)
	body = be32(body, c.InUnknownProtos)
	body = be64(body, c.OutOctets)
	body = be32(body, c.OutUnicastPackets)
	body = be32(body, c.OutMulticast)
	body = be32(body, c.OutBroadcast)
	body = be32(body, c.OutDiscards)
	body = be32(body, c.OutErrors)
	body = be32(body, c.Promiscuous)
	return appendRecord(b, formatIfCounters, body)
}

func (d Datagram) Marshal() []byte {
	b := be32(make([]byte, 0, 1400), Version)
	if v4 := d.Agent.To4(); v4 != nil {
		b = be32(b, addressIPv4)
		b = append(b, v4...)
	} else {
		b = be32(b, addressIPv6)
		b = append(b, d.Agent.To16()...)
	}

	b = be32(b, d.SubAgent)
	b = be32(b, d.Sequence)
	b = be32(b, d.Uptime)
	b = be32(b, uint32(len(d.Samples)))
	for _, s := range d.Samples {
		b = appendRecord(b, s.format(), s.appendBody(nil))
	}

	return b
}

func HeaderLen(agent net.IP) int {
	if agent.To4() != nil {
		return 28
	}

	return 40
}

func EncodedLen(s Sample) int {
	return 8 + len(s.appendBody(nil))
}
