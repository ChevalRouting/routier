package auth

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Put("/api/auth/password", ChangePassword)
}
