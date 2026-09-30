package api

import (
	"context"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	v1 "github.com/ChevalRouting/routier/pkg/server/api/routes/v1"

	friendsroutes "github.com/ChevalRouting/routier/pkg/server/api/routes/friends"
	"github.com/ChevalRouting/routier/pkg/types"

	configroutes "github.com/ChevalRouting/routier/pkg/server/api/routes/config"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/stats"

	"github.com/ChevalRouting/routier/pkg/server/api/workers"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/macros"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/backup"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/announcements"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/dhcp"
	dnsroutes "github.com/ChevalRouting/routier/pkg/server/api/routes/dns"
	natroutes "github.com/ChevalRouting/routier/pkg/server/api/routes/nat"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/routing"

	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/ws"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/auth"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/snapshots"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/system"

	failuresroutes "github.com/ChevalRouting/routier/pkg/server/api/routes/failures"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/setup"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/ha"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/logs"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/apply"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/ui"

	"github.com/ChevalRouting/routier/pkg/server/api/routes/tools"
	"github.com/ChevalRouting/routier/pkg/server/api/routes/wireguard"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router *chi.Mux
	app    *appctx.App
}

func New(ctx context.Context, configPath, dbPath string, jwtSecret []byte, debug bool) (*Server, error) {
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	app := &appctx.App{DB: db, ConfigPath: configPath, JWTSecret: jwtSecret, Validator: appctx.NewValidator(), Debug: debug}
	identityPath := filepath.Join(filepath.Dir(dbPath), identity.KeyFilename)
	if id, err := identity.LoadOrCreate(identityPath); err != nil {
		log.Error().Err(err).Msg("failed to load node identity")
	} else {
		app.Identity = id
	}

	friendcache.SetPath(filepath.Join(filepath.Dir(dbPath), "friends_cache.json"))
	friendcache.Load()
	workers.StartCleanup(ctx, db)
	workers.StartLiveStats()
	workers.StartFriendPoll(configPath)
	return &Server{router: buildRouter(app, nil), app: app}, nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) SetStaticFS(staticFS fs.FS) {
	s.router = buildRouter(s.app, staticFS)
}

func (s *Server) SetAdvertise(port int, tls bool) {
	s.app.AdvertisePort = port
	s.app.AdvertiseTLS = tls
}

func buildRouter(app *appctx.App, staticFS fs.FS) *chi.Mux {
	r := chi.NewRouter()
	r.Use(redactTokenQueryParam)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)
	r.Use(app.Inject())

	r.Post("/api/auth/login", auth.Login)

	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Use(stagingHeaderMiddleware)

		ws.Routes(r)
		auth.Routes(r)
		setup.Routes(r)
		configroutes.Routes(r)
		apply.Routes(r)
		snapshots.Routes(r)
		system.Routes(r)
		failuresroutes.Routes(r)
		backup.Routes(r)
		ha.Routes(r)
		friendsroutes.Routes(r)
		ui.Routes(r)
		wireguard.Routes(r)
		logs.Routes(r)
		stats.Routes(r)
		routing.Routes(r)
		natroutes.Routes(r)
		announcements.Routes(r)
		macros.Routes(r)
		dhcp.Routes(r)
		dnsroutes.Routes(r)
		tools.Routes(r)
	})

	v1.Routes(r, jwtMiddleware)
	docsRoutes(r)

	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api/") {
				types.Err(http.StatusNotFound, "not found").Write(w)
				return
			}

			path := strings.TrimPrefix(req.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			if path == "index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			} else if strings.HasPrefix(path, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}

			f, err := staticFS.Open(path)
			if err != nil {
				if strings.HasPrefix(path, "assets/") {
					http.NotFound(w, req)
					return
				}

				f2, err2 := staticFS.Open("index.html")
				if err2 != nil {
					http.NotFound(w, req)
					return
				}

				_ = f2.Close()
				req2 := req.Clone(req.Context())
				req2.URL.Path = "/"
				fileServer.ServeHTTP(w, req2)
				return
			}

			_ = f.Close()
			fileServer.ServeHTTP(w, req)
		})
	}

	return r
}
