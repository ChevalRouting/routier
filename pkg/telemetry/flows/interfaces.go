package flows

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/telemetry/sflow"
	vnl "github.com/vishvananda/netlink"
)

const ifTypeEthernet = 6

type interfaceNames map[uint32]string

func loadInterfaceNames() interfaceNames {
	names := interfaceNames{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return names
	}

	for _, iface := range ifaces {
		names[uint32(iface.Index)] = iface.Name
	}

	return names
}

func (n interfaceNames) name(index uint32) string {
	if index == 0 {
		return ""
	}

	if name, ok := n[index]; ok {
		return name
	}

	return strconv.FormatUint(uint64(index), 10)
}

func LinkSpeed(device string) uint64 {
	data, err := os.ReadFile(filepath.Join("/sys/class/net", device, "speed"))
	if err != nil {
		return 0
	}

	mbps, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || mbps <= 0 {
		return 0
	}

	return uint64(mbps) * 1_000_000
}

func linkStatus(attrs *vnl.LinkAttrs) uint32 {
	var status uint32
	if attrs.Flags&net.FlagUp != 0 {
		status |= sflow.StatusAdminUp
	}

	if attrs.OperState == vnl.OperUp || attrs.OperState == vnl.OperUnknown && attrs.Flags&net.FlagRunning != 0 {
		status |= sflow.StatusOperUp
	}

	return status
}

func interfaceCounters(device string) (sflow.InterfaceCounters, bool) {
	link, err := vnl.LinkByName(device)
	if err != nil {
		return sflow.InterfaceCounters{}, false
	}

	attrs := link.Attrs()
	c := sflow.InterfaceCounters{
		Index:     uint32(attrs.Index),
		Type:      ifTypeEthernet,
		Speed:     LinkSpeed(device),
		Direction: 1,
		Status:    linkStatus(attrs),
	}

	if attrs.Promisc != 0 {
		c.Promiscuous = 1
	}

	if st := attrs.Statistics; st != nil {
		c.InOctets = st.RxBytes
		c.InUnicastPackets = uint32(st.RxPackets - st.Multicast)
		c.InMulticastPackets = uint32(st.Multicast)
		c.InDiscards = uint32(st.RxDropped)
		c.InErrors = uint32(st.RxErrors)
		c.OutOctets = st.TxBytes
		c.OutUnicastPackets = uint32(st.TxPackets)
		c.OutDiscards = uint32(st.TxDropped)
		c.OutErrors = uint32(st.TxErrors)
	}

	return c, true
}
