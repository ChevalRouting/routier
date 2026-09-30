package announcements

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const maxBodySize = 1 << 16

var validLevels = map[string]bool{"info": true, "warning": true, "danger": true}

// List godoc
// @Summary  List all announcements
// @Tags announcements
// @Produce json
// @Success 200 {object} types.Response[[]types.Announcement]
// @Security BearerAuth
// @Router /api/announcements [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	items, err := webdb.ListAnnouncements(r.Context(), app.DB)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list announcements"))
		return
	}

	types.OK(w, items)
}

// Active godoc
// @Summary  List enabled announcements (for the banner)
// @Tags announcements
// @Produce json
// @Success 200 {object} types.Response[[]types.Announcement]
// @Security BearerAuth
// @Router /api/announcements/active [get]
func Active(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	items, err := webdb.EnabledAnnouncements(r.Context(), app.DB)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list announcements"))
		return
	}

	types.OK(w, items)
}

// Create godoc
// @Summary  Create an announcement
// @Tags announcements
// @Produce json
// @Param body body types.Announcement true "announcement"
// @Success 200 {object} types.Response[types.Announcement]
// @Security BearerAuth
// @Router /api/announcements [post]
func Create(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	a, appErr := decode(r)
	if appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	id, err := webdb.CreateAnnouncement(requests.DurableContext(r), app.DB, a)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "create announcement"))
		return
	}

	a.ID = id
	types.OK(w, a)
}

// Update godoc
// @Summary  Update an announcement
// @Tags announcements
// @Produce json
// @Param id path int true "announcement id"
// @Param body body types.Announcement true "announcement"
// @Success 200 {object} types.Response[types.Announcement]
// @Security BearerAuth
// @Router /api/announcements/{id} [put]
func Update(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		types.Err(http.StatusBadRequest, "invalid id").Write(w)
		return
	}

	a, appErr := decode(r)
	if appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	a.ID = id
	if err := webdb.UpdateAnnouncement(requests.DurableContext(r), app.DB, a); err != nil {
		if errors.Is(err, webdb.ErrAnnouncementNotFound) {
			types.Err(http.StatusNotFound, "announcement not found").Write(w)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "update announcement"))
		return
	}

	types.OK(w, a)
}

// Delete godoc
// @Summary  Delete an announcement
// @Tags announcements
// @Produce json
// @Param id path int true "announcement id"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/announcements/{id} [delete]
func Delete(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		types.Err(http.StatusBadRequest, "invalid id").Write(w)
		return
	}

	if err := webdb.DeleteAnnouncement(requests.DurableContext(r), app.DB, id); err != nil {
		if errors.Is(err, webdb.ErrAnnouncementNotFound) {
			types.Err(http.StatusNotFound, "announcement not found").Write(w)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "delete announcement"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func decode(r *http.Request) (*types.Announcement, *types.AppError) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		return nil, types.Errorf(http.StatusBadRequest, "failed to read body")
	}

	var a types.Announcement
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, types.Errorf(http.StatusBadRequest, "invalid announcement: %v", err)
	}

	a.Message = strings.TrimSpace(a.Message)
	if a.Message == "" {
		return nil, types.Errorf(http.StatusBadRequest, "message is required")
	}

	if a.Level == "" {
		a.Level = "info"
	}

	if !validLevels[a.Level] {
		return nil, types.Errorf(http.StatusBadRequest, "level must be info, warning or danger")
	}

	return &a, nil
}
