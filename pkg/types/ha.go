package types

type VRRPPeer struct {
	IP       string `json:"ip"`
	Priority int    `json:"priority"`
	LastSeen string `json:"last_seen,omitempty" validate:"optional"`
}

type VRRPInstanceStatus struct {
	Name      string     `json:"name,omitempty" validate:"optional"`
	Interface string     `json:"interface"`
	ID        int        `json:"id"`
	VIPs      []string   `json:"vips,omitempty" validate:"optional"`
	Priority  int        `json:"priority"`
	State     string     `json:"state"`
	MasterIP  string     `json:"master_ip,omitempty" validate:"optional"`
	Peers     []VRRPPeer `json:"peers,omitempty" validate:"optional"`
}

type ConntrackdStatus struct {
	Running   bool             `json:"running"`
	Entries   int64            `json:"entries"`
	ByProto   map[string]int64 `json:"by_proto,omitempty" validate:"optional"`
	TCPStates map[string]int64 `json:"tcp_states,omitempty" validate:"optional"`
}

type HAStatusResponse struct {
	VRRP       []VRRPInstanceStatus `json:"vrrp,omitempty" validate:"optional"`
	Conntrackd *ConntrackdStatus    `json:"conntrackd,omitempty" validate:"optional"`
}
