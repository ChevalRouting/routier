package types

type SetupStatus struct {
	NeedsPasswordChange bool `json:"needs_password_change"`
	OnboardingComplete  bool `json:"onboarding_complete"`
	HasInterfaces       bool `json:"has_interfaces"`
}

type SystemNic struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac,omitempty" validate:"optional"`
	Operstate string   `json:"operstate,omitempty" validate:"optional"`
	Addrs     []string `json:"addrs,omitempty" validate:"optional"`
	Physical  bool     `json:"physical,omitempty" validate:"optional"`
	Driver    string   `json:"driver,omitempty" validate:"optional"`
	Speed     int      `json:"speed,omitempty" validate:"optional"`
	Duplex    string   `json:"duplex,omitempty" validate:"optional"`
	Carrier   bool     `json:"carrier,omitempty" validate:"optional"`
	PCIVendor string   `json:"pci_vendor,omitempty" validate:"optional"`
	PCIDevice string   `json:"pci_device,omitempty" validate:"optional"`
}

type SetupHost struct {
	DefaultIface string   `json:"default_iface,omitempty" validate:"optional"`
	Gateway4     string   `json:"gateway4,omitempty" validate:"optional"`
	DhcpIfaces   []string `json:"dhcp_ifaces"`
	Nameservers  []string `json:"nameservers"`
	TakenOver    bool     `json:"taken_over"`
}

type InstallStatus struct {
	Live          bool     `json:"live"`
	InstallerOnly bool     `json:"installer_only"`
	State         string   `json:"state" enums:"idle,running,done,failed"`
	Disk          string   `json:"disk,omitempty" validate:"optional"`
	Error         string   `json:"error,omitempty" validate:"optional"`
	Log           []string `json:"log"`
}

type InstallRequest struct {
	Disk     string `json:"disk" validate:"required"`
	Hostname string `json:"hostname" validate:"required,hostname"`
	Timezone string `json:"timezone" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}
