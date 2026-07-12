package dhcp

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/dhcp", Overview)
	r.Get("/api/dhcp/stats", Stats)
	r.Get("/api/dhcp/leases/stream", LeaseStream)
	r.Get("/api/dhcp/subnets", Subnets)
	r.Get("/api/dhcp/leases", Leases)
	r.Get("/api/dhcp/free-ip", FreeIP)
	r.Post("/api/dhcp/leases/clear", ClearLease)
	r.Post("/api/dhcp/reservations/from-lease", ReserveFromLease)

	r.Post("/api/dhcp/reservations", AddReservation)
	r.Post("/api/dhcp/reservations/del", DelReservation)
	r.Post("/api/dhcp/reservations/persist", PersistLease)
}
