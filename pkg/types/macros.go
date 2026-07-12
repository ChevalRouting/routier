package types

type DiffLine struct {
	Type string `json:"type" enums:"add,remove,same"`
	Text string `json:"text"`
}

type FileDiff struct {
	File   string     `json:"file"`
	Status string     `json:"status" enums:"added,removed,modified"`
	Lines  []DiffLine `json:"lines,omitempty" validate:"optional"`
}

type MacroInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Sections    []string `json:"sections,omitempty" validate:"optional"`
	CreatedAt   string   `json:"created_at"`
	CreatedBy   string   `json:"created_by"`
	ApplyCount  int      `json:"apply_count"`
	AppliedAt   *string  `json:"applied_at,omitempty" validate:"optional"`
}

type ConflictDetail struct {
	Section string `json:"section"`
	Key     string `json:"key,omitempty" validate:"optional"`
	Message string `json:"message"`
}

type MacroConflictsResponse struct {
	Conflicts []ConflictDetail `json:"conflicts,omitempty" validate:"optional"`
}

type CreateMacroRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MacroApplyRequest struct {
	Force bool `json:"force"`
}
