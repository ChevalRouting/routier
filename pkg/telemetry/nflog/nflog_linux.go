//go:build linux

package nflog

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

const (
	receiveBuffer = 4 << 20
	readTimeout   = 500 * time.Millisecond
)

type Conn struct {
	fd  int
	seq uint32
	buf []byte
}

func Open(group uint16, snaplen uint32) (*Conn, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_NETFILTER)
	if err != nil {
		return nil, fmt.Errorf("nflog socket: %w", err)
	}

	c := &Conn{fd: fd, buf: make([]byte, 1<<16)}
	if err := c.setup(group, snaplen); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	return c, nil
}

func (c *Conn) setup(group uint16, snaplen uint32) error {
	if err := unix.Bind(c.fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return fmt.Errorf("nflog bind: %w", err)
	}

	if err := unix.SetsockoptInt(c.fd, unix.SOL_SOCKET, unix.SO_RCVBUFFORCE, receiveBuffer); err != nil {
		_ = unix.SetsockoptInt(c.fd, unix.SOL_SOCKET, unix.SO_RCVBUF, receiveBuffer)
	}

	timeout := unix.NsecToTimeval(readTimeout.Nanoseconds())
	if err := unix.SetsockoptTimeval(c.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout); err != nil {
		return fmt.Errorf("nflog read timeout: %w", err)
	}

	for _, family := range []uint8{unix.AF_INET, unix.AF_INET6} {
		_ = c.request(family, 0, commandAttr(cmdPFBind))
	}

	if err := c.request(unix.AF_UNSPEC, group, commandAttr(cmdBind)); err != nil {
		return fmt.Errorf("nflog bind group %d: %w", group, err)
	}

	if err := c.request(unix.AF_UNSPEC, group, copyModeAttr(snaplen)); err != nil {
		return fmt.Errorf("nflog copy mode: %w", err)
	}

	return nil
}

func (c *Conn) request(family uint8, group uint16, attrs []byte) error {
	c.seq++
	if err := unix.Sendto(c.fd, configMessage(c.seq, family, group, attrs), 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return err
	}

	buf := make([]byte, unix.Getpagesize())
	n, _, err := unix.Recvfrom(c.fd, buf, 0)
	if err != nil {
		return err
	}

	return parseMessages(buf[:n], ignorePacket)
}

func (c *Conn) Read(onPacket func(Packet)) (overrun bool, err error) {
	n, _, err := unix.Recvfrom(c.fd, c.buf, 0)
	if errors.Is(err, unix.ENOBUFS) {
		return true, nil
	}

	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return false, parseMessages(c.buf[:n], onPacket)
}

func (c *Conn) Close() error {
	return unix.Close(c.fd)
}
