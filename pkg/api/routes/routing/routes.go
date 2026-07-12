package routing

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/routing/learned", Learned)
	r.Get("/api/routing/routes", KernelRoutes)
	r.Get("/api/routing/neighbors", Neighbors)
}
