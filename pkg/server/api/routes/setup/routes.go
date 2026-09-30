package setup

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/setup/status", Status)
	r.Post("/api/setup/complete", Complete)
	r.Get("/api/setup/onboarding", GetOnboarding)
	r.Put("/api/setup/onboarding", PutOnboarding)
	r.Get("/api/system/nics", Nics)
}
