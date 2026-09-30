package macros

import (
	"net/http"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

func macroToInfo(m *webdb.Macro) types.MacroInfo {
	info := types.MacroInfo{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Sections:    m.Sections,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		CreatedBy:   m.CreatedBy,
		ApplyCount:  m.ApplyCount,
	}

	if info.Sections == nil {
		info.Sections = []string{}
	}

	if m.AppliedAt != nil {
		s := m.AppliedAt.Format(time.RFC3339)
		info.AppliedAt = &s
	}

	return info
}

// List godoc
// @Summary  List saved macros
// @Tags macros
// @Produce json
// @Success 200 {object} types.Response[[]types.MacroInfo]
// @Security BearerAuth
// @Router /api/macros [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	macros, err := webdb.ListMacros(r.Context(), app.DB)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to list macros"))
		return
	}

	infos := make([]types.MacroInfo, len(macros))
	for i, m := range macros {
		infos[i] = macroToInfo(m)
	}

	types.OK(w, infos)
}
