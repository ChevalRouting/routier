package apply

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/apply/pending", Pending)
	r.Post("/api/apply/confirm", Confirm)
	r.Get("/api/apply/logs", Logs)
	r.Get("/api/apply/logs/{id}", Log)
}
