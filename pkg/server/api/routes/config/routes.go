package config

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/config", GetConfig)
	r.Get("/api/config/schema", GetSchema)
	r.Get("/api/config/schema/version", GetSchemaVersion)
	r.Get("/api/config/diff", GetDiff)
	r.Get("/api/config/render-diff", RenderDiff)
	r.Get("/api/config/nftables/vars", GetNftablesVars)
	r.Post("/api/config/nftables/validate", ValidateNftables)
	r.Put("/api/config/import", Import)
	r.Put("/api/config/raw", PutRaw)
	r.Get("/api/config/{section}", GetSection)
	r.Put("/api/config/{section}", PutSection)
	r.Delete("/api/config/staging", DiscardStaging)
	r.Post("/api/config/apply", Apply)
}
