package snapshots

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
)

// @Summary Read a saved artifact from a snapshot
// @Tags snapshots
// @Produce json
// @Param id path string true "snapshot id"
// @Param dest query string true "saved artifact destination path"
// @Success 200 {object} types.Response[types.ArtifactContent]
// @Security BearerAuth
// @Router /api/snapshots/{id}/artifact [get]
func Artifact(w http.ResponseWriter, r *http.Request) {
	dest := r.URL.Query().Get("dest")
	content, err := apply.ReadSnapshotArtifact(chi.URLParam(r, "id"), dest)
	if err != nil {
		types.ErrNotFound.Write(w)
		return
	}

	types.OK(w, types.ArtifactContent{Dest: dest, Content: content})
}
