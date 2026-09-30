package friends

import (
	"encoding/base64"
	"net/http"
	"sort"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/keepalived"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Hello godoc
// @Summary  Friend hello/handshake
// @Tags     friends
// @Produce  json
// @Param    challenge query string false "challenge nonce to sign"
// @Success  200 {object} types.Response[types.FriendsHello]
// @Security BearerAuth
// @Router   /api/friends/hello [get]
func Hello(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	cfg, _ := cfgpkg.Load(app.ConfigPath)
	resp := buildHello(app, cfg)

	if challenge := r.URL.Query().Get("challenge"); challenge != "" && app.Identity != nil {
		resp.Signature = base64.StdEncoding.EncodeToString(app.Identity.Sign([]byte(challenge)))
	}

	types.OK(w, resp)
}

func buildHello(app *appctx.App, cfg *cfgpkg.Config) types.FriendsHello {
	resp := types.FriendsHello{
		Version:   appctx.Version,
		OS:        osInfo(),
		UptimeSec: int64(time.Since(appctx.StartTime).Seconds()),
	}

	if app.Identity != nil {
		resp.IdentityFingerprint = app.Identity.Fingerprint()
		resp.IdentityPublicKey = app.Identity.PublicKeyBase64()
		resp.X25519PublicKey = app.Identity.X25519PublicBase64()
		resp.Encryption = true
	}

	if cfg != nil {
		resp.Hostname = cfg.Hostname
		resp.Conntrackd = cfg.HA != nil && cfg.HA.Conntrackd != nil
	}

	resp.ConntrackdRunning = svc.ServiceRunning("conntrackd")
	for instance, st := range keepalived.States() {
		resp.VRRP = append(resp.VRRP, types.FriendVRRPRole{Instance: instance, State: st.State})
	}

	sort.Slice(resp.VRRP, func(i, j int) bool { return resp.VRRP[i].Instance < resp.VRRP[j].Instance })

	return resp
}
