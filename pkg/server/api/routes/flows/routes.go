package flows

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/flows", List)
	r.Get("/api/flows/status", Status)
	r.Get("/api/flows/top", Top)
	r.Get("/api/flows/series", Series)
	r.Get("/api/flows/sankey", Sankey)
	r.Get("/api/flows/metadata", Metadata)
}
