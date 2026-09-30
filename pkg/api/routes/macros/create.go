package macros

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/requests"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/macro"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Create godoc
// @Summary  Save current staged changes as a macro
// @Tags macros
// @Produce json
// @Param body body types.CreateMacroRequest true "macro name/description"
// @Success 200 {object} types.Response[types.MacroInfo]
// @Security BearerAuth
// @Router /api/macros [post]
func Create(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	var req types.CreateMacroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		types.Err(http.StatusBadRequest, "invalid body: "+err.Error()).Write(w)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		types.Err(http.StatusBadRequest, "name is required").Write(w)
		return
	}

	cfgstore.Mu.RLock()
	base, err := config.Load(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	stagingPath := cfgstore.StagingPath(app.ConfigPath, username)
	cfgstore.Mu.RLock()
	_, statErr := os.Stat(stagingPath)
	cfgstore.Mu.RUnlock()
	if os.IsNotExist(statErr) {
		types.Err(http.StatusBadRequest, "no pending changes to save as macro").Write(w)
		return
	}

	mod, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read staging config"))
		return
	}

	sections := macro.ComputeSections(base, mod)
	if len(sections) == 0 {
		types.Err(http.StatusBadRequest, "staged config is identical to live config").Write(w)
		return
	}

	baseYAML, _ := yaml.Marshal(base)
	modYAML, _ := yaml.Marshal(mod)

	m := &webdb.Macro{
		ID:          webdb.NewMacroID(),
		Name:        req.Name,
		Description: req.Description,
		BaseYAML:    string(baseYAML),
		ModYAML:     string(modYAML),
		Sections:    sections,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   username,
	}

	if err := webdb.InsertMacro(requests.DurableContext(r), app.DB, m); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			types.Err(http.StatusConflict, "macro \""+req.Name+"\" already exists").Write(w)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to save macro"))
		return
	}

	types.OK(w, macroToInfo(m))
}
