package tools

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/tools/subnet", Subnet)
	r.Get("/api/tools/reverse", Reverse)
	r.Get("/api/tools/range", Range)
}
