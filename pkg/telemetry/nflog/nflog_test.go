package nflog

import (
	"encoding/binary"
	"errors"
	"syscall"
	"testing"
)

func packetMessage(attrs ...[]byte) []byte {
	var body []byte
	for _, attr := range attrs {
		body = append(body, attr...)
	}

	msg := configMessage(1, 2, 7, body)
	binary.NativeEndian.PutUint16(msg[4:6], subsysULOG<<8|msgPacket)
	return msg
}

func be32(v uint32) []byte {
	return binary.BigEndian.AppendUint32(nil, v)
}

func TestParseMessagesDecodesPackets(t *testing.T) {
	payload := []byte{0x45, 0, 0, 20, 1, 2, 3}
	header := []byte{0x08, 0x00, 1, 0}
	msg := packetMessage(attribute(attrPacketHdr, header), attribute(attrInDev, be32(3)), attribute(attrOutDev, be32(5)), attribute(attrPayload, payload))
	msg = append(msg, msg...)

	var got []Packet
	if err := parseMessages(msg, func(p Packet) { got = append(got, p) }); err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("parsed %d packets, want 2", len(got))
	}

	p := got[0]
	if p.Protocol != 0x0800 || p.InDev != 3 || p.OutDev != 5 || string(p.Payload) != string(payload) {
		t.Fatalf("packet = %+v", p)
	}
}

func TestParseMessagesReportsNetlinkErrors(t *testing.T) {
	msg := make([]byte, nlmsgHdrLen+4)
	binary.NativeEndian.PutUint32(msg[0:4], uint32(len(msg)))
	binary.NativeEndian.PutUint16(msg[4:6], nlmsgError)
	code := -int32(syscall.EPERM)
	binary.NativeEndian.PutUint32(msg[16:20], uint32(code))

	if err := parseMessages(msg, ignorePacket); !errors.Is(err, syscall.EPERM) {
		t.Fatalf("err = %v, want EPERM", err)
	}
}

func TestParseMessagesRejectsTruncatedInput(t *testing.T) {
	msg := packetMessage(attribute(attrInDev, be32(3)))
	if err := parseMessages(msg[:len(msg)-2], ignorePacket); !errors.Is(err, ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated", err)
	}
}

func TestCopyModeAttrEncodesSnaplen(t *testing.T) {
	attr := copyModeAttr(128)
	if binary.NativeEndian.Uint16(attr[2:4]) != cfgAttrMode || binary.BigEndian.Uint32(attr[4:8]) != 128 || attr[8] != copyPacket {
		t.Fatalf("copy mode attr = %x", attr)
	}
}
