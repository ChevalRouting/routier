package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Add godoc
// @Summary  Add a friend
// @Tags friends
// @Produce json
// @Param body body types.AddFriendRequest true "friend"
// @Success 200 {object} types.Response[types.FriendInfo]
// @Security BearerAuth
// @Router /api/friends [post]
func Add(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	var req types.AddFriendRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	if req.Name == "" || req.URL == "" || req.Token == "" {
		types.Err(http.StatusBadRequest, "name, url and token are required").Write(w)
		return
	}

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if friendspkg.Index(cfg, req.Name) >= 0 {
		types.Err(http.StatusConflict, "friend \""+req.Name+"\" already exists").Write(w)
		return
	}

	f := &cfgpkg.Friend{
		Name:          req.Name,
		URL:           req.URL,
		Token:         req.Token,
		TLSSkipVerify: req.TLSSkipVerify,
		Enabled:       req.Enabled,
	}

	client := friendspkg.NewClient(req.URL, req.Token, req.TLSSkipVerify)
	if hello, herr := client.Hello(r.Context()); herr != nil {
		log.Warn().Err(herr).Str("friend", f.Name).Msg("could not reach friend at add time; added unverified")
	} else if self := selfFingerprint(app); self != "" && identity.FingerprintMatch(self, hello.IdentityFingerprint) {
		types.Err(http.StatusBadRequest, "refusing to add self as a friend").Write(w)
		return
	} else {
		f.Hostname = hello.Hostname
		f.Identity.Fingerprint = hello.IdentityFingerprint
		f.Identity.PublicKey = hello.IdentityPublicKey
	}

	if err := friendspkg.Add(cfg, f); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "add friend"))
		return
	}

	if err := cfgstore.WriteLive(app.ConfigPath, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save config"))
		return
	}

	types.OK(w, friendInfo(f))
}
