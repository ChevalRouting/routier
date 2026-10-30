package types

type PortMode string

const (
	PortAccess    PortMode = "access"
	PortTrunk     PortMode = "trunk"
	PortBondSlave PortMode = "bond-member"
	PortUnbridged PortMode = "unbridged"
)

type SwitchFacts struct {
	Identity string `json:"identity"`
	Model    string `json:"model"`
	Serial   string `json:"serial"`
	Version  string `json:"version"`
}

type SwitchPort struct {
	Name     string   `json:"name"`
	Label    string   `json:"label,omitempty" validate:"optional"`
	Type     string   `json:"type"`
	Running  bool     `json:"running"`
	Disabled bool     `json:"disabled"`
	MAC      string   `json:"mac"`
	Rate     string   `json:"rate"`
	Comment  string   `json:"comment"`
	Bridge   string   `json:"bridge"`
	Mode     PortMode `json:"mode"`
	PVID     int      `json:"pvid"`
	Untagged []int    `json:"untagged"`
	Tagged   []int    `json:"tagged"`
	Bond     string   `json:"bond"`
}

type SwitchBond struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
	Mode    string   `json:"mode"`
}

type SwitchVLAN struct {
	ID       int      `json:"id"`
	Bridge   string   `json:"bridge"`
	Tagged   []string `json:"tagged"`
	Untagged []string `json:"untagged"`
	Dynamic  bool     `json:"dynamic"`
}

type SwitchState struct {
	Facts         SwitchFacts  `json:"facts"`
	VLANFiltering bool         `json:"vlan_filtering"`
	Bridges       []string     `json:"bridges"`
	Ports         []SwitchPort `json:"ports"`
	Bonds         []SwitchBond `json:"bonds"`
	VLANs         []SwitchVLAN `json:"vlans"`
}

type CredentialField struct {
	Key      string `json:"key" yaml:"key"`
	Label    string `json:"label" yaml:"label"`
	Type     string `json:"type" yaml:"type"`
	Default  string `json:"default_value" yaml:"default"`
	Required bool   `json:"required" yaml:"required"`
	Help     string `json:"help" yaml:"help"`
}

type FaceplateGroup struct {
	Group string   `json:"group"`
	Kind  string   `json:"kind"`
	Rows  int      `json:"rows"`
	Order string   `json:"order"`
	Ports []string `json:"ports"`
}

type SwitchModel struct {
	ID          string            `json:"id"`
	Vendor      string            `json:"vendor"`
	Name        string            `json:"name"`
	Driver      string            `json:"driver"`
	Generic     bool              `json:"generic"`
	Credentials []CredentialField `json:"credentials"`
	Faceplate   []FaceplateGroup  `json:"faceplate"`
}

type SwitchStatus struct {
	Name           string       `json:"name"`
	Model          SwitchModel  `json:"model"`
	Host           string       `json:"host"`
	Interface      string       `json:"interface"`
	Uplink         string       `json:"uplink"`
	HasCredentials bool         `json:"has_credentials"`
	Reachable      bool         `json:"reachable"`
	Error          string       `json:"error,omitempty" validate:"optional"`
	ReadAt         int64        `json:"read_at"`
	State          *SwitchState `json:"state,omitempty" validate:"optional"`
}

type SwitchCandidate struct {
	ChassisID    string   `json:"chassis_id"`
	Name         string   `json:"name"`
	SysDescr     string   `json:"sys_descr"`
	MgmtIP       string   `json:"mgmt_ip"`
	Capabilities string   `json:"capabilities"`
	ModelID      string   `json:"model_id,omitempty" validate:"optional"`
	LocalIface   string   `json:"local_iface"`
	PortID       string   `json:"port_id"`
	Protocols    []string `json:"protocols"`
}

type SwitchCredentialsRequest struct {
	Values map[string]string `json:"values" validate:"required"`
}

type SwitchPlanPreview struct {
	Steps []string `json:"steps"`
}
