package auth

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Put("/api/auth/password", ChangePassword)
	r.Post("/api/auth/password-hash", HashPassword)
	r.Get("/api/auth/api-keys", ListAPIKeys)
	r.Post("/api/auth/api-keys", CreateAPIKey)
	r.Delete("/api/auth/api-keys/{id}", RevokeAPIKey)
}
