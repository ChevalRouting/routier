package types

type StatusResponse struct {
	Status string `json:"status"`
}

type ApplyResult struct {
	Status   string          `json:"status"`
	SnapID   string          `json:"snapID,omitempty" validate:"optional"`
	Warning  string          `json:"warning,omitempty" validate:"optional"`
	BundleID string          `json:"bundleID,omitempty" validate:"optional"`
	Errors   []ArtifactError `json:"errors,omitempty" validate:"optional"`
}

type ApplyPendingResponse struct {
	Pending   bool   `json:"pending"`
	SnapID    string `json:"snap_id,omitempty" validate:"optional"`
	Timeout   int    `json:"timeout,omitempty" validate:"optional"`
	Remaining int    `json:"remaining,omitempty" validate:"optional"`
}

type ApplyLogRecord struct {
	ID           string  `json:"id"`
	Source       string  `json:"source"`
	ConfigPath   string  `json:"config_path,omitempty" validate:"optional"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   *string `json:"finished_at,omitempty" validate:"optional"`
	SnapID       string  `json:"snap_id,omitempty" validate:"optional"`
	Result       string  `json:"result,omitempty" validate:"optional"`
	HasBundle    bool    `json:"has_bundle,omitempty" validate:"optional"`
	ConfirmedAt  *string `json:"confirmed_at,omitempty" validate:"optional"`
	RolledBackAt *string `json:"rolledback_at,omitempty" validate:"optional"`
}

type ApplyLogsResponse struct {
	Records []ApplyLogRecord `json:"records,omitempty" validate:"optional"`
}

type SnapshotInfo struct {
	ID    string   `json:"id"`
	Time  string   `json:"time"`
	Files []string `json:"files,omitempty" validate:"optional"`
}

type SnapshotsResponse struct {
	Snapshots []SnapshotInfo `json:"snapshots,omitempty" validate:"optional"`
}
