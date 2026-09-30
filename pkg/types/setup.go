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
