package api

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	v1 "github.com/ChevalRouting/routier/pkg/api/routes/v1"

	friendsroutes "github.com/ChevalRouting/routier/pkg/api/routes/friends"
	"github.com/ChevalRouting/routier/pkg/types"

	configroutes "github.com/ChevalRouting/routier/pkg/api/routes/config"

	"github.com/ChevalRouting/routier/pkg/api/routes/stats"

	"github.com/ChevalRouting/routier/pkg/api/workers"

	"github.com/ChevalRouting/routier/pkg/api/routes/macros"

	"github.com/ChevalRouting/routier/pkg/api/routes/backup"

	"github.com/ChevalRouting/routier/pkg/api/routes/announcements"
	"github.com/ChevalRouting/routier/pkg/api/routes/dhcp"
	natroutes "github.com/ChevalRouting/routier/pkg/api/routes/nat"
	"github.com/ChevalRouting/routier/pkg/api/routes/routing"

	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/api/routes/ws"

	"github.com/ChevalRouting/routier/pkg/api/routes/auth"

	"github.com/ChevalRouting/routier/pkg/api/routes/snapshots"

	failuresroutes "github.com/ChevalRouting/routier/pkg/api/routes/failures"

	"github.com/ChevalRouting/routier/pkg/api/routes/setup"

	"github.com/ChevalRouting/routier/pkg/api/routes/ha"
	"github.com/ChevalRouting/routier/pkg/api/routes/logs"

	"github.com/ChevalRouting/routier/pkg/api/routes/apply"
	"github.com/ChevalRouting/routier/pkg/api/routes/ui"

	"github.com/ChevalRouting/routier/pkg/api/routes/tools"
	"github.com/ChevalRouting/routier/pkg/api/routes/wireguard"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"

	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router *chi.Mux
	app    *appctx.App
}

func New(configPath, dbPath string, jwtSecret []byte, debug bool) (*Server, error) {
	db, err := webdb.InitDB(dbPath)
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
	workers.StartCleanup(db)
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

			f, err := staticFS.Open(path)
			if err != nil {
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
