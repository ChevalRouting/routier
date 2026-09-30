package types

type RoutesResponse struct {
	Routes []KernelRoute `json:"routes,omitempty" validate:"optional"`
	Total  int           `json:"total"`
}

type KernelRoute struct {
	Dst      string `json:"dst"`
	Gateway  string `json:"gateway,omitempty" validate:"optional"`
	Dev      string `json:"dev,omitempty" validate:"optional"`
	Protocol string `json:"protocol"`
	Metric   int    `json:"metric,omitempty" validate:"optional"`
	Family   string `json:"family"`
}

type Neighbor struct {
	Dst    string `json:"dst"`
	Dev    string `json:"dev"`
	LLAddr string `json:"lladdr,omitempty" validate:"optional"`
	State  string `json:"state"`
	Family string `json:"family"`
}

type NeighborsResponse struct {
	Neighbors []Neighbor `json:"neighbors,omitempty" validate:"optional"`
}

type LLDPNeighbor struct {
	LocalIface   string `json:"local_iface"`
	Protocol     string `json:"protocol"`
	ChassisID    string `json:"chassis_id,omitempty" validate:"optional"`
	ChassisName  string `json:"chassis_name,omitempty" validate:"optional"`
	SysDescr     string `json:"sys_descr,omitempty" validate:"optional"`
	MgmtIP       string `json:"mgmt_ip,omitempty" validate:"optional"`
	PortID       string `json:"port_id,omitempty" validate:"optional"`
	PortDescr    string `json:"port_descr,omitempty" validate:"optional"`
	Capabilities string `json:"capabilities,omitempty" validate:"optional"`
	VLAN         string `json:"vlan,omitempty" validate:"optional"`
	Age          string `json:"age,omitempty" validate:"optional"`
}

type LLDPNeighborsResponse struct {
	Neighbors []LLDPNeighbor `json:"neighbors,omitempty" validate:"optional"`
}

type LearnedRoute struct {
	Prefix   string `json:"prefix"`
	Protocol string `json:"protocol"`
	NextHop  string `json:"nexthop,omitempty" validate:"optional"`
	Iface    string `json:"interface,omitempty" validate:"optional"`
	Metric   int    `json:"metric,omitempty" validate:"optional"`
	Distance int    `json:"distance,omitempty" validate:"optional"`
}

type LearnedRoutesResponse struct {
	OSPF []LearnedRoute `json:"ospf,omitempty" validate:"optional"`
	BGP  []LearnedRoute `json:"bgp,omitempty" validate:"optional"`
}
