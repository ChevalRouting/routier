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
