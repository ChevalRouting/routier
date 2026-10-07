package snapshots

import (
	"net/http"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  List config snapshots
// @Tags snapshots
// @Produce json
// @Success 200 {object} types.Response[types.SnapshotsResponse]
// @Security BearerAuth
// @Router /api/snapshots [get]
func List(w http.ResponseWriter, _ *http.Request) {
	infos, err := apply.ListSnapshotInfos()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to list snapshots"))
		return
	}

	out := make([]types.SnapshotInfo, 0, len(infos))
	for _, s := range infos {
		out = append(out, types.SnapshotInfo{
			ID:    s.ID,
			Time:  s.Time.Format(time.RFC3339),
			Files: s.Files,
		})
	}

	types.OK(w, types.SnapshotsResponse{Snapshots: out})
}
