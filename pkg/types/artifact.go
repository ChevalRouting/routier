package types

type ArtifactError struct {
	Tool     string `json:"tool,omitempty" validate:"optional"`
	Artifact string `json:"artifact,omitempty" validate:"optional"`
	Dest     string `json:"dest,omitempty" validate:"optional"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty" validate:"optional"`
	Message  string `json:"message"`
}

type RenderValidateResult struct {
	OK     bool            `json:"ok"`
	Errors []ArtifactError `json:"errors,omitempty" validate:"optional"`
}
