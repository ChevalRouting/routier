package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// PairFriend godoc
// @Summary  Validate a friend's fingerprint and complete pairing
// @Tags     friends
// @Produce  json
// @Param    name path string true "friend name"
// @Param    body body types.VerifyFriendRequest false "operator-confirmed fingerprint"
// @Success  200 {object} types.Response[types.FriendInfo]
// @Security BearerAuth
// @Router   /api/friends/{name}/pair [post]
func PairFriend(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	if app.Identity == nil {
		types.Err(http.StatusConflict, "this node has no identity; cannot pair").Write(w)
		return
	}

	var req types.VerifyFriendRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
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

	client := friendspkg.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	hello, err := client.Hello(r.Context())
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to contact friend"))
		return
	}

	if identity.FingerprintMatch(app.Identity.Fingerprint(), hello.IdentityFingerprint) {
		types.Err(http.StatusBadRequest, "refusing to pair with self").Write(w)
		return
	}

	if req.Fingerprint != "" && !identity.FingerprintMatch(req.Fingerprint, hello.IdentityFingerprint) {
		types.Err(http.StatusConflict, "fingerprint changed since you reviewed it (possible MITM)").Write(w)
		return
	}

	if f.Identity.Fingerprint != "" && !identity.FingerprintMatch(f.Identity.Fingerprint, hello.IdentityFingerprint) {
		types.Err(http.StatusConflict, "friend fingerprint changed since it was added (possible MITM)").Write(w)
		return
	}

	if err := pairFriend(r.Context(), app, cfg, f); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "pair friend"))
		return
	}

	if err := cfgstore.WriteLive(app.ConfigPath, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save config"))
		return
	}

	types.OK(w, friendInfo(f))
}
