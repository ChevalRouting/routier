package system

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/system/version", Version)
	r.Get("/api/system/bonds", Bonds)
	r.Get("/api/system/interfaces/{name}", InterfaceStatus)
	r.Get("/api/system/updates", Updates)
	r.Get("/api/system/upgrade", UpgradeStatus)
	r.Post("/api/system/upgrade", StartUpgrade)
	r.Post("/api/system/reboot", Reboot)
	r.Post("/api/system/shutdown", Shutdown)
}
