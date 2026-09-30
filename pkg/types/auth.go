package types

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ChangePasswordRequest struct {
	Current string `json:"current" validate:"required"`
	New     string `json:"new" validate:"required,min=1"`
}

type PasswordHashRequest struct {
	Password string `json:"password" validate:"required,min=1"`
}

type PasswordHashResponse struct {
	Hash string `json:"hash"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name" validate:"required,min=1,max=128"`
}

type APIKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	CreatedAt int64  `json:"created_at"`
	CreatedBy string `json:"created_by"`
	RevokedAt *int64 `json:"revoked_at,omitempty"`
}

type CreateAPIKeyResponse struct {
	APIKey APIKey `json:"api_key"`
	Token  string `json:"token"`
}
