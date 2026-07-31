package types

type FailureRecord struct {
	ID        string          `json:"id"`
	Time      string          `json:"time"`
	Source    string          `json:"source,omitempty" validate:"optional"`
	SnapID    string          `json:"snap_id,omitempty" validate:"optional"`
	Errors    []ArtifactError `json:"errors,omitempty" validate:"optional"`
	Artifacts []string        `json:"artifacts,omitempty" validate:"optional"`
}

type FailuresResponse struct {
	Failures []FailureRecord `json:"failures,omitempty" validate:"optional"`
}

type ArtifactContent struct {
	Dest    string `json:"dest"`
	Content string `json:"content"`
}
