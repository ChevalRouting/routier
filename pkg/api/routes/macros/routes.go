package macros

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/macros", List)
	r.Post("/api/macros", Create)
	r.Get("/api/macros/{id}", Get)
	r.Delete("/api/macros/{id}", Delete)
	r.Get("/api/macros/{id}/diff", Diff)
	r.Post("/api/macros/{id}/apply", Apply)
}
