package flows

import "encoding/binary"

const (
	ethernetHeaderLen = 14
	ethernetFCSLen    = 4
	etherTypeIPv4     = 0x0800
	etherTypeIPv6     = 0x86dd
	maxSampledHeader  = 128
)

func etherType(family uint8) uint16 {
	if family == 6 {
		return etherTypeIPv6
	}

	return etherTypeIPv4
}

func ethernetFrame(hwHeader []byte, family uint8, payload []byte) []byte {
	frame := make([]byte, ethernetHeaderLen, ethernetHeaderLen+len(payload))
	if len(hwHeader) == ethernetHeaderLen {
		copy(frame, hwHeader[:12])
	}

	binary.BigEndian.PutUint16(frame[12:14], etherType(family))
	frame = append(frame, payload...)
	return frame[:min(len(frame), maxSampledHeader)]
}
