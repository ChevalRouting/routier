package types

type IfaceStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	TxErrors  uint64 `json:"tx_errors"`
	OperState string `json:"operstate"`
}

type ProtoStats struct {
	TCPActiveOpens  int64 `json:"tcp_active_opens"`
	TCPPassiveOpens int64 `json:"tcp_passive_opens"`
	TCPAttemptFails int64 `json:"tcp_attempt_fails"`
	TCPEstabResets  int64 `json:"tcp_estab_resets"`
	TCPCurrEstab    int64 `json:"tcp_curr_estab"`
	TCPInSegs       int64 `json:"tcp_in_segs"`
	TCPOutSegs      int64 `json:"tcp_out_segs"`
	TCPRetransSegs  int64 `json:"tcp_retrans_segs"`
	UDPInDatagrams  int64 `json:"udp_in_datagrams"`
	UDPOutDatagrams int64 `json:"udp_out_datagrams"`
	UDPInErrors     int64 `json:"udp_in_errors"`
	UDPNoPorts      int64 `json:"udp_no_ports"`
	ICMPInMsgs      int64 `json:"icmp_in_msgs"`
	ICMPOutMsgs     int64 `json:"icmp_out_msgs"`
}

type BGPPeerSummary struct {
	Address     string `json:"address"`
	ASN         int    `json:"asn"`
	State       string `json:"state"`
	Uptime      string `json:"uptime"`
	Description string `json:"description,omitempty" validate:"optional"`
	MsgRcvd     int    `json:"msg_rcvd"`
	MsgSent     int    `json:"msg_sent"`
	Prefixes    int    `json:"prefixes"`
}

type BGPStats struct {
	RouterID string           `json:"router_id"`
	LocalASN int              `json:"local_asn"`
	Peers    []BGPPeerSummary `json:"peers,omitempty" validate:"optional"`
}

type OSPFNeighborSummary struct {
	NeighborID string `json:"neighbor_id"`
	Interface  string `json:"interface"`
	State      string `json:"state"`
	Address    string `json:"address"`
	Area       string `json:"area,omitempty" validate:"optional"`
}

type OSPFStats struct {
	RouterID  string                `json:"router_id"`
	Neighbors []OSPFNeighborSummary `json:"neighbors,omitempty" validate:"optional"`
}

type SystemStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemTotal   uint64  `json:"mem_total"`
	MemUsed    uint64  `json:"mem_used"`
	MemBuffers uint64  `json:"mem_buffers"`
	MemCached  uint64  `json:"mem_cached"`
	SwapTotal  uint64  `json:"swap_total"`
	SwapUsed   uint64  `json:"swap_used"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	Uptime     uint64  `json:"uptime_seconds"`
	Processes  int     `json:"processes"`
}

type StatsResponse struct {
	Interfaces map[string]*IfaceStats `json:"interfaces,omitempty" validate:"optional"`
	BGP        *BGPStats              `json:"bgp,omitempty" validate:"optional"`
	OSPF       *OSPFStats             `json:"ospf,omitempty" validate:"optional"`
	System     *SystemStats           `json:"system,omitempty" validate:"optional"`
}

type IfaceHistoryPoint struct {
	TS        int64    `json:"ts"`
	RxBytes   int64    `json:"rx_bytes"`
	TxBytes   int64    `json:"tx_bytes"`
	RxPkts    int64    `json:"rx_pkts"`
	TxPkts    int64    `json:"tx_pkts"`
	RxErrs    int64    `json:"rx_errs"`
	TxErrs    int64    `json:"tx_errs"`
	RxBps     *float64 `json:"rx_bps"`
	TxBps     *float64 `json:"tx_bps"`
	RxPps     *float64 `json:"rx_pps"`
	TxPps     *float64 `json:"tx_pps"`
	OperState string   `json:"operstate"`
}

type SystemHistoryPoint struct {
	TS       int64   `json:"ts"`
	CPUPct   float64 `json:"cpu_pct"`
	MemUsed  int64   `json:"mem_used"`
	MemTotal int64   `json:"mem_total"`
	Load1    float64 `json:"load1"`
}

type BGPHistoryPoint struct {
	TS       int64  `json:"ts"`
	State    string `json:"state"`
	Uptime   string `json:"uptime"`
	MsgRcvd  int64  `json:"msg_rcvd"`
	MsgSent  int64  `json:"msg_sent"`
	Prefixes int64  `json:"prefixes"`
}

type ProtoHistoryPoint struct {
	TS              int64 `json:"ts"`
	TCPActiveOpens  int64 `json:"tcp_active_opens"`
	TCPPassiveOpens int64 `json:"tcp_passive_opens"`
	TCPAttemptFails int64 `json:"tcp_attempt_fails"`
	TCPEstabResets  int64 `json:"tcp_estab_resets"`
	TCPCurrEstab    int64 `json:"tcp_curr_estab"`
	TCPInSegs       int64 `json:"tcp_in_segs"`
	TCPOutSegs      int64 `json:"tcp_out_segs"`
	TCPRetransSegs  int64 `json:"tcp_retrans_segs"`
	UDPInDatagrams  int64 `json:"udp_in_datagrams"`
	UDPOutDatagrams int64 `json:"udp_out_datagrams"`
	UDPInErrors     int64 `json:"udp_in_errors"`
	UDPNoPorts      int64 `json:"udp_no_ports"`
	ICMPInMsgs      int64 `json:"icmp_in_msgs"`
	ICMPOutMsgs     int64 `json:"icmp_out_msgs"`
}

type IfaceTotalPoint struct {
	TS    int64    `json:"ts"`
	RxBps *float64 `json:"rx_bps"`
	TxBps *float64 `json:"tx_bps"`
	RxPps *float64 `json:"rx_pps"`
	TxPps *float64 `json:"tx_pps"`
}

type IfaceUsagePoint struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

type UsageSummary struct {
	PerIface map[string]IfaceUsagePoint `json:"per_iface,omitempty" validate:"optional"`
	Total    IfaceUsagePoint            `json:"total"`
}

type StatsHistoryResponse struct {
	Interfaces map[string][]IfaceHistoryPoint `json:"interfaces,omitempty" validate:"optional"`
	BGPPeers   map[string][]BGPHistoryPoint   `json:"bgp_peers,omitempty" validate:"optional"`
	Proto      []ProtoHistoryPoint            `json:"proto,omitempty" validate:"optional"`
	System     []SystemHistoryPoint           `json:"system,omitempty" validate:"optional"`
	Total      []IfaceTotalPoint              `json:"total,omitempty" validate:"optional"`
	Usage      *UsageSummary                  `json:"usage,omitempty" validate:"optional"`
}

type NeighborStat struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac,omitempty" validate:"optional"`
	Dev      string `json:"dev"`
	State    string `json:"state"`
	Family   string `json:"family"`
	Hostname string `json:"hostname,omitempty" validate:"optional"`
}

type NeighborStatsResponse struct {
	Neighbors []NeighborStat `json:"neighbors,omitempty" validate:"optional"`
	TS        int64          `json:"ts"`
}

type ProcessInfo struct {
	PID    int     `json:"pid"`
	Name   string  `json:"name"`
	State  string  `json:"state"`
	CPUPct float64 `json:"cpu_pct"`
	MemRSS uint64  `json:"mem_rss"`
	Cmd    string  `json:"cmd,omitempty" validate:"optional"`
}

type ProcessesResponse struct {
	Processes []ProcessInfo `json:"processes,omitempty" validate:"optional"`
}
