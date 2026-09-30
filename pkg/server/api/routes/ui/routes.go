package ui

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/ui/layout", GetLayout)
	r.Put("/api/ui/layout", PutLayout)
}
