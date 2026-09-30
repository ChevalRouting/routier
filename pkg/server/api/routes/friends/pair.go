package friends

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Pair godoc
// @Summary  Complete identity pairing (friend-token)
// @Tags friends
// @Produce json
// @Param body body types.FriendPairRequest true "pairing payload"
// @Success 200 {object} types.Response[types.FriendPairResponse]
// @Security BearerAuth
// @Router /api/friends/pair [post]
func Pair(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	name, ok := appctx.FriendUsername(r.Context())
	if !ok {
		types.Err(http.StatusForbidden, "pairing requires a friend token").Write(w)
		return
	}

	var req types.FriendPairRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	pub, err := identity.ParsePublicKey(req.IdentityPublicKey)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid identity public key"))
		return
	}

	if identity.Fingerprint(pub) != req.IdentityFingerprint {
		types.Err(http.StatusBadRequest, "identity public key does not match fingerprint").Write(w)
		return
	}

	sig, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil || !identity.Verify(pub, []byte(req.IdentityFingerprint), sig) {
		types.Err(http.StatusUnauthorized, "identity signature verification failed").Write(w)
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

	if err := friendspkg.PinIdentity(f, req.IdentityFingerprint, req.IdentityPublicKey, req.X25519PublicKey); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusConflict, err, "pin identity"))
		return
	}

	if req.Hostname != "" {
		f.Hostname = req.Hostname
	}

	if req.Port > 0 {
		if host, _, herr := net.SplitHostPort(r.RemoteAddr); herr == nil {
			scheme := req.Scheme
			if scheme == "" {
				scheme = "http"
			}

			f.URL = fmt.Sprintf("%s://%s:%d", scheme, formatHost(host), req.Port)
			if scheme == "https" {
				f.TLSSkipVerify = true
			}
		}
	}

	if err := cfgstore.WriteLive(app.ConfigPath, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save config"))
		return
	}

	resp := types.FriendPairResponse{Hostname: cfg.Hostname, Paired: true}
	if app.Identity != nil {
		resp.IdentityFingerprint = app.Identity.Fingerprint()
		resp.IdentityPublicKey = app.Identity.PublicKeyBase64()
		resp.X25519PublicKey = app.Identity.X25519PublicBase64()
	}

	types.OK(w, resp)
}
