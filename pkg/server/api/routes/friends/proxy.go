package friends

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func Proxy(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	if _, isFriend := appctx.FriendUsername(r.Context()); isFriend {
		types.Err(http.StatusForbidden, "friend tokens may not proxy").Write(w)
		return
	}

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	f := friendspkg.Get(cfg, name)
	if f == nil {
		types.Err(http.StatusNotFound, "friend \""+name+"\" not found").Write(w)
		return
	}

	target, err := url.Parse(strings.TrimRight(f.URL, "/"))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "invalid friend url"))
		return
	}

	rest := "/" + strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	token := f.Token

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: f.TLSSkipVerify}

	rp := &httputil.ReverseProxy{
		Transport:      transport,
		Director:       func(req *http.Request) { proxyCallback(target, rest, token, req) },
		ModifyResponse: proxyHandler,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "friend \""+name+"\" unreachable"))
		},
	}

	rp.ServeHTTP(w, r)
}

func proxyHandler(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized {
		resp.StatusCode = http.StatusBadGateway
		resp.Status = http.StatusText(http.StatusBadGateway)
	}

	return nil
}

func proxyCallback(target *url.URL, rest string, token string, req *http.Request) {
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = target.Host
	req.URL.Path = target.Path + rest
	req.Header.Set("Authorization", "Bearer "+token)
}
