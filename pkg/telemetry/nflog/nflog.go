package nflog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
)

const (
	subsysULOG = 4

	msgPacket = 0
	msgConfig = 1

	cfgAttrCmd  = 1
	cfgAttrMode = 2

	cmdBind   = 1
	cmdPFBind = 3

	copyPacket = 2

	attrPacketHdr = 1
	attrInDev     = 4
	attrOutDev    = 5
	attrPayload   = 9
	attrHwHeader  = 16

	nlmsgHdrLen = 16
	nfgenHdrLen = 4
	nlaHdrLen   = 4
	nlaTypeMask = 0x3fff
	nlmsgError  = 2
	nlmsgDone   = 3
	flagRequest = 0x1
	flagAck     = 0x4
)

var ErrTruncated = errors.New("nflog: truncated netlink message")

type Packet struct {
	Protocol uint16
	InDev    uint32
	OutDev   uint32
	HwHeader []byte
	Payload  []byte
}

func ignorePacket(Packet) {}

func align4(n int) int {
	return (n + 3) &^ 3
}

func configMessage(seq uint32, family uint8, group uint16, attrs []byte) []byte {
	length := nlmsgHdrLen + nfgenHdrLen + len(attrs)
	msg := make([]byte, length)
	binary.NativeEndian.PutUint32(msg[0:4], uint32(length))
	binary.NativeEndian.PutUint16(msg[4:6], subsysULOG<<8|msgConfig)
	binary.NativeEndian.PutUint16(msg[6:8], flagRequest|flagAck)
	binary.NativeEndian.PutUint32(msg[8:12], seq)
	msg[16] = family
	binary.BigEndian.PutUint16(msg[18:20], group)
	copy(msg[nlmsgHdrLen+nfgenHdrLen:], attrs)
	return msg
}

func attribute(kind uint16, value []byte) []byte {
	length := nlaHdrLen + len(value)
	attr := make([]byte, align4(length))
	binary.NativeEndian.PutUint16(attr[0:2], uint16(length))
	binary.NativeEndian.PutUint16(attr[2:4], kind)
	copy(attr[nlaHdrLen:], value)
	return attr
}

func commandAttr(command uint8) []byte {
	return attribute(cfgAttrCmd, []byte{command})
}

func copyModeAttr(snaplen uint32) []byte {
	value := make([]byte, 6)
	binary.BigEndian.PutUint32(value[0:4], snaplen)
	value[4] = copyPacket
	return attribute(cfgAttrMode, value)
}

func parsePacket(body []byte) (Packet, error) {
	var p Packet
	for len(body) >= nlaHdrLen {
		length := int(binary.NativeEndian.Uint16(body[0:2]))
		if length < nlaHdrLen || length > len(body) {
			return p, ErrTruncated
		}

		value := body[nlaHdrLen:length]
		switch binary.NativeEndian.Uint16(body[2:4]) & nlaTypeMask {
		case attrPacketHdr:
			if len(value) >= 2 {
				p.Protocol = binary.BigEndian.Uint16(value[0:2])
			}
		case attrInDev:
			if len(value) >= 4 {
				p.InDev = binary.BigEndian.Uint32(value[0:4])
			}
		case attrOutDev:
			if len(value) >= 4 {
				p.OutDev = binary.BigEndian.Uint32(value[0:4])
			}
		case attrPayload:
			p.Payload = value
		case attrHwHeader:
			p.HwHeader = value
		}

		body = body[min(align4(length), len(body)):]
	}

	return p, nil
}

func parseMessages(buf []byte, onPacket func(Packet)) error {
	for len(buf) >= nlmsgHdrLen {
		length := int(binary.NativeEndian.Uint32(buf[0:4]))
		if length < nlmsgHdrLen || length > len(buf) {
			return ErrTruncated
		}

		kind := binary.NativeEndian.Uint16(buf[4:6])
		body := buf[nlmsgHdrLen:length]
		switch {
		case kind == nlmsgError:
			if err := netlinkError(body); err != nil {
				return err
			}
		case kind == nlmsgDone:
		case kind == subsysULOG<<8|msgPacket && len(body) >= nfgenHdrLen:
			packet, err := parsePacket(body[nfgenHdrLen:])
			if err != nil {
				return err
			}

			onPacket(packet)
		}

		buf = buf[min(align4(length), len(buf)):]
	}

	return nil
}

func netlinkError(body []byte) error {
	if len(body) < 4 {
		return ErrTruncated
	}

	code := int32(binary.NativeEndian.Uint32(body[0:4]))
	if code == 0 {
		return nil
	}

	return fmt.Errorf("nflog: %w", syscall.Errno(-code))
}
