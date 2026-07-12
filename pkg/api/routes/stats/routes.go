package stats

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/stats", Get)
	r.Get("/api/stats/history", History)
	r.Get("/api/stats/neighbors", Neighbors)
	r.Get("/api/stats/processes", Processes)
}
