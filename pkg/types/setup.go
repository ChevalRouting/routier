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
}
