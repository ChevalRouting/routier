package announcements

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/announcements", List)
	r.Get("/api/announcements/active", Active)
	r.Post("/api/announcements", Create)
	r.Put("/api/announcements/{id}", Update)
	r.Delete("/api/announcements/{id}", Delete)
}
