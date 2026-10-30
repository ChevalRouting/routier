package snapshots

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/snapshots", List)
	r.Get("/api/snapshots/{id}/artifact", Artifact)
	r.Post("/api/snapshots/{id}/restore", Restore)
}
