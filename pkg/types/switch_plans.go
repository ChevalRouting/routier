package types

type ChangeKind string

const (
	ChangeSetAccess  ChangeKind = "set-access"
	ChangeSetTrunk   ChangeKind = "set-trunk"
	ChangeCreateBond ChangeKind = "create-bond"
	ChangeDeleteBond ChangeKind = "delete-bond"
)

type PortChange struct {
	Kind   ChangeKind `json:"kind" validate:"required,oneof=set-access set-trunk create-bond delete-bond"`
	Ports  []string   `json:"ports" validate:"optional"`
	VLAN   int        `json:"vlan" validate:"optional,min=0,max=4094"`
	Tagged []int      `json:"tagged" validate:"optional,dive,min=1,max=4094"`
	Bond   string     `json:"bond" validate:"optional,max=32"`
	Mode   string     `json:"mode" validate:"optional,oneof=802.3ad active-backup balance-xor balance-rr broadcast balance-tlb balance-alb"`
}

type SwitchPlan struct {
	Changes []PortChange `json:"changes" validate:"required,min=1,dive"`
	Force   bool         `json:"force" validate:"optional"`
}

type PlanPreview struct {
	Steps    []string `json:"steps"`
	Warnings []string `json:"warnings"`
}
