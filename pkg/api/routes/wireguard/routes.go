package wireguard

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Post("/api/wireguard/keygen", Keygen)
	r.Post("/api/wireguard/pubkey", Pubkey)
}
