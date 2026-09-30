package nat

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/nat", List)
	r.Put("/api/nat", Replace)
}
