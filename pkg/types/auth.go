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
