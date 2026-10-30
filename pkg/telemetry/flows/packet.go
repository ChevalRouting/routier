package flows

import (
	"encoding/binary"
	"net/netip"
)

const (
	protoTCP    = 6
	protoUDP    = 17
	protoSCTP   = 132
	ipv6HdrLen  = 40
	minIPv4Len  = 20
	extHopByHop = 0
	extRouting  = 43
	extFragment = 44
	extAuth     = 51
	extDestOpts = 60
)

type Key struct {
	InDev   uint32
	OutDev  uint32
	Family  uint8
	Proto   uint8
	Src     netip.Addr
	Dst     netip.Addr
	SrcPort uint16
	DstPort uint16
}

type Sampled struct {
	Key    Key
	Length uint32
}

func hasPorts(proto uint8) bool {
	return proto == protoTCP || proto == protoUDP || proto == protoSCTP
}

func readPorts(key *Key, l4 []byte) {
	if hasPorts(key.Proto) && len(l4) >= 4 {
		key.SrcPort = binary.BigEndian.Uint16(l4[0:2])
		key.DstPort = binary.BigEndian.Uint16(l4[2:4])
	}
}

func parseIPv4(b []byte) (Sampled, bool) {
	if len(b) < minIPv4Len {
		return Sampled{}, false
	}

	headerLen := int(b[0]&0x0f) * 4
	if headerLen < minIPv4Len || len(b) < headerLen {
		return Sampled{}, false
	}

	key := Key{Family: 4, Proto: b[9]}
	key.Src, _ = netip.AddrFromSlice(b[12:16])
	key.Dst, _ = netip.AddrFromSlice(b[16:20])

	fragmentOffset := binary.BigEndian.Uint16(b[6:8]) & 0x1fff
	if fragmentOffset == 0 {
		readPorts(&key, b[headerLen:])
	}

	return Sampled{Key: key, Length: uint32(binary.BigEndian.Uint16(b[2:4]))}, true
}

func parseIPv6(b []byte) (Sampled, bool) {
	if len(b) < ipv6HdrLen {
		return Sampled{}, false
	}

	key := Key{Family: 6}
	key.Src, _ = netip.AddrFromSlice(b[8:24])
	key.Dst, _ = netip.AddrFromSlice(b[24:40])
	length := ipv6HdrLen + uint32(binary.BigEndian.Uint16(b[4:6]))

	next, rest := b[6], b[ipv6HdrLen:]
	for {
		switch next {
		case extHopByHop, extRouting, extDestOpts:
			if len(rest) < 2 || len(rest) < (int(rest[1])+1)*8 {
				key.Proto = next
				return Sampled{Key: key, Length: length}, true
			}

			next, rest = rest[0], rest[(int(rest[1])+1)*8:]
		case extAuth:
			if len(rest) < 2 || len(rest) < (int(rest[1])+2)*4 {
				key.Proto = next
				return Sampled{Key: key, Length: length}, true
			}

			next, rest = rest[0], rest[(int(rest[1])+2)*4:]
		case extFragment:
			if len(rest) < 8 {
				key.Proto = next
				return Sampled{Key: key, Length: length}, true
			}

			key.Proto = rest[0]
			if binary.BigEndian.Uint16(rest[2:4])&0xfff8 == 0 {
				readPorts(&key, rest[8:])
			}

			return Sampled{Key: key, Length: length}, true
		default:
			key.Proto = next
			readPorts(&key, rest)
			return Sampled{Key: key, Length: length}, true
		}
	}
}

func ParsePacket(payload []byte, inDev, outDev uint32) (Sampled, bool) {
	if len(payload) == 0 {
		return Sampled{}, false
	}

	var s Sampled
	var ok bool
	switch payload[0] >> 4 {
	case 4:
		s, ok = parseIPv4(payload)
	case 6:
		s, ok = parseIPv6(payload)
	}

	s.Key.InDev, s.Key.OutDev = inDev, outDev
	return s, ok
}
