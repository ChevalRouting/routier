package friends

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Preview a prospective friend's identity
// @Tags friends
// @Produce json
// @Param body body types.FriendPreviewRequest true "url and token"
// @Success 200 {object} types.Response[types.FriendPreview]
// @Security BearerAuth
// @Router /api/friends/preview [post]
func Preview(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	var req types.FriendPreviewRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	if req.URL == "" || req.Token == "" {
		types.Err(http.StatusBadRequest, "url and token are required").Write(w)
		return
	}

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	client := friendspkg.NewClient(req.URL, req.Token, req.TLSSkipVerify)
	types.OK(w, friendspkg.Preview(r.Context(), client, cfg, selfFingerprint(app)))
}
