package dns

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/dns", Overview)
	r.Get("/api/dns/stats", Stats)
	r.Get("/api/dns/queries/stream", QueryStream)
	r.Get("/api/dns/zones", Zones)
	r.Get("/api/dns/zones/{name}", Zone)
	r.Post("/api/dns/zones/{name}/reload", ReloadZone)
	r.Post("/api/dns/cache/flush", FlushCache)
	r.Post("/api/dns/restart", Restart)
	r.Post("/api/dns/query", Query)
	r.Get("/api/dns/ddns", DDNS)
	r.Delete("/api/dns/ddns/{zone}/records", DeleteDDNSRecord)
}
