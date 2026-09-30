package failures

import (
	"net/http"
	"time"

	failurespkg "github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func Routes(r chi.Router) {
	r.Get("/api/failures", List)
	r.Get("/api/failures/{id}", Get)
	r.Get("/api/failures/{id}/artifact", Artifact)
	r.Get("/api/failures/{id}/export", Export)
	r.Delete("/api/failures/{id}", Remove)
}

func toRecord(m failurespkg.Meta) types.FailureRecord {
	return types.FailureRecord{
		ID:        m.ID,
		Time:      m.Time.Format(time.RFC3339),
		Source:    m.Source,
		SnapID:    m.SnapID,
		Errors:    m.Errors,
		Artifacts: m.Artifacts,
	}
}

// List godoc
// @Summary  List preserved failed-apply bundles
// @Tags failures
// @Produce json
// @Success 200 {object} types.Response[types.FailuresResponse]
// @Security BearerAuth
// @Router /api/failures [get]
func List(w http.ResponseWriter, _ *http.Request) {
	metas := failurespkg.List()
	out := make([]types.FailureRecord, 0, len(metas))
	for _, m := range metas {
		out = append(out, toRecord(m))
	}

	types.OK(w, types.FailuresResponse{Failures: out})
}

// Get godoc
// @Summary  Get a failed-apply bundle
// @Tags failures
// @Produce json
// @Param id path string true "bundle id"
// @Success 200 {object} types.Response[types.FailureRecord]
// @Security BearerAuth
// @Router /api/failures/{id} [get]
func Get(w http.ResponseWriter, r *http.Request) {
	m, err := failurespkg.Get(chi.URLParam(r, "id"))
	if err != nil {
		types.ErrNotFound.Write(w)
		return
	}

	types.OK(w, toRecord(m))
}

// Artifact godoc
// @Summary  Read a rendered artifact from a failed-apply bundle
// @Tags failures
// @Produce json
// @Param id path string true "bundle id"
// @Param dest query string true "rendered artifact destination path"
// @Success 200 {object} types.Response[types.ArtifactContent]
// @Security BearerAuth
// @Router /api/failures/{id}/artifact [get]
func Artifact(w http.ResponseWriter, r *http.Request) {
	dest := r.URL.Query().Get("dest")
	data, err := failurespkg.ReadArtifact(chi.URLParam(r, "id"), dest)
	if err != nil {
		types.ErrNotFound.Write(w)
		return
	}

	types.OK(w, types.ArtifactContent{Dest: dest, Content: string(data)})
}

// Export godoc
// @Summary  Download a failed-apply bundle as a tar.gz
// @Tags failures
// @Produce octet-stream
// @Param id path string true "bundle id"
// @Param redact query bool false "redact secrets (default true)"
// @Success 200 {string} string "bundle archive"
// @Security BearerAuth
// @Router /api/failures/{id}/export [get]
func Export(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	redact := true
	if v := r.URL.Query().Get("redact"); v == "false" || v == "0" {
		redact = false
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=routier-failure-"+id+".tar.gz")
	if err := failurespkg.Export(id, w, redact); err != nil {
		log.Warn().Err(err).Str("id", id).Msg("failure export failed")
	}
}

// Remove godoc
// @Summary  Delete a failed-apply bundle
// @Tags failures
// @Produce json
// @Param id path string true "bundle id"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/failures/{id} [delete]
func Remove(w http.ResponseWriter, r *http.Request) {
	if err := failurespkg.Remove(chi.URLParam(r, "id")); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to remove bundle"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "removed"})
}
