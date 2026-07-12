package kea

import "encoding/json"

type response struct {
	Result    int             `json:"result"`
	Text      string          `json:"text"`
	Arguments json.RawMessage `json:"arguments"`
}

type Subnet struct {
	ID     int    `json:"id"`
	Subnet string `json:"subnet"`
}

type Lease struct {
	IPAddress string `json:"ip-address"`
	HWAddress string `json:"hw-address"`
	DUID      string `json:"duid"`
	Hostname  string `json:"hostname"`
	SubnetID  int    `json:"subnet-id"`
	State     int    `json:"state"`
	Expire    int64  `json:"expire"`
}

type leaseList struct {
	Leases []Lease `json:"leases"`
}

type Host struct {
	IPAddress   string   `json:"ip-address"`
	IPAddresses []string `json:"ip-addresses"`
	HWAddress   string   `json:"hw-address"`
	DUID        string   `json:"duid"`
	Hostname    string   `json:"hostname"`
}

type hostList struct {
	Hosts []Host `json:"hosts"`
}

type reqIP struct {
	IPAddress string `json:"ip-address"`
}

type reqSubnet struct {
	SubnetID int `json:"subnet-id"`
}

type reqLeaseSubnet struct {
	Subnets []int `json:"subnets"`
}

type reqSubnetIP struct {
	SubnetID  int    `json:"subnet-id"`
	IPAddress string `json:"ip-address"`
}

type reservationV4 struct {
	SubnetID  int    `json:"subnet-id"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip-address"`
	HWAddress string `json:"hw-address,omitempty" validate:"optional"`
	DUID      string `json:"duid,omitempty" validate:"optional"`
}

type reservationV6 struct {
	SubnetID    int      `json:"subnet-id"`
	Hostname    string   `json:"hostname"`
	IPAddresses []string `json:"ip-addresses"`
	DUID        string   `json:"duid"`
}

type reqReservationV4 struct {
	Reservation reservationV4 `json:"reservation"`
}

type reqReservationV6 struct {
	Reservation reservationV6 `json:"reservation"`
}

type commandBase struct {
	Command   string          `json:"command"`
	Service   []string        `json:"service"`
	Arguments json.RawMessage `json:"arguments,omitempty" validate:"optional"`
}
