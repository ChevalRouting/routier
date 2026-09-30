package backup

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/backup/export", Export)
	r.Post("/api/backup/import", Import)
}
