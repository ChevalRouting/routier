package friends

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/api/friends", List)
	r.Post("/api/friends", Add)
	r.Post("/api/friends/preview", Preview)
	r.Get("/api/friends/status", Status)
	r.Post("/api/friends/{name}/check", Check)
	r.Post("/api/friends/{name}/pair", PairFriend)
	r.Get("/api/friends/{name}/config", RemoteConfig)
	r.Get("/api/friends/{name}/nftables", NftablesImport)
	r.Get("/api/friends/{name}/cache", Cache)
	r.Get("/api/friends/hello", Hello)
	r.Get("/api/friends/poll", Poll)
	r.Post("/api/friends/pair", Pair)
	r.Post("/api/friends/sync", SyncAll)
	r.Get("/api/friends/interfaces", Interfaces)
	r.Get("/api/friends/exports", Exports)
	r.Put("/api/friends/{name}", Update)
	r.Delete("/api/friends/{name}", Delete)
	r.Post("/api/friends/{name}/sync", Sync)
	r.Get("/api/friends/{name}/interfaces", RemoteInterfaces)
	r.Post("/api/friends/{name}/wireguard", DeriveWireguard)
	r.Post("/api/friends/{name}/tunnel", DeriveTunnel)
	r.Post("/api/friends/{name}/vrrp", ConfigureVRRP)
	r.Post("/api/friends/{name}/conntrack", ConfigureConntrack)
	r.HandleFunc("/api/friends/{name}/proxy/*", Proxy)
}
