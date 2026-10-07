package friends

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Batched, shared-key-encrypted friend poll
// @Tags friends
// @Produce json
// @Param    have query string false "config hash the caller already has"
// @Success  200 {object} types.Response[types.FriendPoll]
// @Security BearerAuth
// @Router   /api/friends/poll [get]
func Poll(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	name, ok := appctx.FriendUsername(r.Context())
	if !ok {
		types.Err(http.StatusForbidden, "poll requires a friend token").Write(w)
		return
	}

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if app.Identity == nil {
		types.Err(http.StatusConflict, "no local identity").Write(w)
		return
	}

	f := friendspkg.Get(cfg, name)
	if f == nil || f.Identity.X25519PublicKey == "" {
		types.Err(http.StatusConflict, "friend \""+name+"\" not paired for encryption").Write(w)
		return
	}

	key, err := app.Identity.SharedKey(f.Identity.X25519PublicKey)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "derive shared key"))
		return
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal config"))
		return
	}

	sum := sha256.Sum256(cfgJSON)
	hash := hex.EncodeToString(sum[:])

	poll := types.FriendPoll{
		Hello:      buildHello(app, cfg),
		Interfaces: friendspkg.InterfaceAddresses(cfg),
		Exports:    exportVars(cfg),
		ConfigHash: hash,
	}

	if r.URL.Query().Get("have") != hash {
		poll.Config = cfgJSON
	}

	env, err := json.Marshal(types.Response[types.FriendPoll]{Result: &poll})
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal poll"))
		return
	}

	sealed, err := identity.SealShared(key, env)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "seal poll"))
		return
	}

	w.Header().Set("Content-Type", types.SharedSealedContentType)
	_, _ = w.Write(sealed)
}

func exportVars(cfg *cfgpkg.Config) []types.FriendVar {
	cfgpkg.ResolveInterfaces(cfg)
	vars := render.NftVars(cfg)
	out := make([]types.FriendVar, 0, len(vars))
	for _, v := range vars {
		out = append(out, types.FriendVar{Name: v.Name, Value: v.Value})
	}

	return out
}
