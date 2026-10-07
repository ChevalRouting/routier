package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Routes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1", func(r chi.Router) { routesCallback(authMiddleware, r) })
}

func routesHandler(r chi.Router) {
	r.Use(v1SessionMiddleware)

	r.Get("/", handleV1GetSession)
	r.Delete("/", handleV1DeleteSession)
	r.Get("/diff", handleV1SessionDiff)
	r.Post("/validate", handleV1SessionValidate)
	r.Post("/apply", handleV1SessionApply)
	r.Get("/config", handleV1GetSessionConfig)
	r.Put("/config", handleV1PutSessionConfig)

	r.Get("/hostname", handleV1GetHostname)
	r.Put("/hostname", handleV1PutHostname)

	r.Get("/sysctl", handleV1GetSysctl)
	r.Put("/sysctl", handleV1PutSysctl)
	r.Get("/sysctl/{key}", handleV1GetSysctlKey)
	r.Put("/sysctl/{key}", handleV1PutSysctlKey)
	r.Delete("/sysctl/{key}", handleV1DeleteSysctlKey)

	r.Get("/dns", handleV1GetDNS)
	r.Put("/dns", handleV1PutDNS)
	r.Get("/routing", handleV1GetRouting)
	r.Put("/routing", handleV1PutRouting)
	r.Get("/logging", handleV1GetLogging)
	r.Put("/logging", handleV1PutLogging)
	r.Get("/ssh", handleV1GetSSH)
	r.Put("/ssh", handleV1PutSSH)
	r.Get("/ha", handleV1GetHA)
	r.Put("/ha", handleV1PutHA)
	r.Get("/gai", handleV1GetGAI)
	r.Put("/gai", handleV1PutGAI)
	r.Get("/monitoring", handleV1GetMonitoring)
	r.Put("/monitoring", handleV1PutMonitoring)

	v1MapRoutes(r, "interfaces", handleV1GetInterfaces, handleV1PutInterfaces, handleV1GetInterface, handleV1PutInterface, handleV1DeleteInterface)
	v1MapRoutes(r, "tunnels", handleV1GetTunnels, handleV1PutTunnels, handleV1GetTunnel, handleV1PutTunnel, handleV1DeleteTunnel)
	v1MapRoutes(r, "wireguard", handleV1GetWireguards, handleV1PutWireguards, handleV1GetWireguard, handleV1PutWireguard, handleV1DeleteWireguard)
	v1MapRoutes(r, "users", handleV1GetUsers, handleV1PutUsers, handleV1GetUser, handleV1PutUser, handleV1DeleteUser)
	v1MapRoutes(r, "services", handleV1GetServices, handleV1PutServices, handleV1GetService, handleV1PutService, handleV1DeleteService)
	v1MapRoutes(r, "vrfs", handleV1GetVRFs, handleV1PutVRFs, handleV1GetVRF, handleV1PutVRF, handleV1DeleteVRF)

	r.Get("/nftables", handleV1GetNftables)
	r.Put("/nftables", handleV1PutNftables)
	r.Get("/boot_modules", handleV1GetBootModules)
	r.Put("/boot_modules", handleV1PutBootModules)
}

func routesCallback(authMiddleware func(http.Handler) http.Handler, r chi.Router) {
	r.Group(func(r chi.Router) { routesCallbackCallback(authMiddleware, r) })
}

func routesCallbackCallback(authMiddleware func(http.Handler) http.Handler, r chi.Router) {
	r.Use(authMiddleware)

	r.Get("/config", handleV1GetConfig)
	r.Post("/sessions", handleV1CreateSession)
	r.Get("/sessions", handleV1ListSessions)

	r.Route("/sessions/{sessionID}", routesHandler)
}
