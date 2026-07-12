package types

type Announcement struct {
	ID          int64  `json:"id"`
	Message     string `json:"message"`
	Level       string `json:"level" enums:"info,warning,danger"`
	Enabled     bool   `json:"enabled"`
	Dismissible bool   `json:"dismissible"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}
