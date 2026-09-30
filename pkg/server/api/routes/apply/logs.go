package apply

import (
	"net/http"
	"time"

	"github.com/ChevalRouting/routier/pkg/applylog"
	"github.com/ChevalRouting/routier/pkg/types"
)

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}

	s := t.Format(time.RFC3339)
	return &s
}

// Logs godoc
// @Summary  List apply-history records
// @Tags apply
// @Produce json
// @Success 200 {object} types.Response[types.ApplyLogsResponse]
// @Security BearerAuth
// @Router /api/apply/logs [get]
func Logs(w http.ResponseWriter, _ *http.Request) {
	recs := applylog.List()
	out := make([]types.ApplyLogRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, types.ApplyLogRecord{
			ID:           r.ID,
			Source:       r.Source,
			ConfigPath:   r.ConfigPath,
			StartedAt:    r.StartedAt.Format(time.RFC3339),
			FinishedAt:   rfc3339Ptr(r.FinishedAt),
			SnapID:       r.SnapID,
			Result:       r.Result,
			HasBundle:    r.HasBundle,
			ConfirmedAt:  rfc3339Ptr(r.ConfirmedAt),
			RolledBackAt: rfc3339Ptr(r.RolledBackAt),
		})
	}

	types.OK(w, types.ApplyLogsResponse{Records: out})
}
