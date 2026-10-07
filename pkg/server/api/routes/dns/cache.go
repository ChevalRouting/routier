package dns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type flushRequest struct {
	Name string `json:"name,omitempty" validate:"optional"`
}

type queryRequest struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty" validate:"optional"`
}

// @Summary  Flush the resolver cache, or one name within it
// @Tags dns
// @Accept json
// @Produce json
// @Param request body dns.flushRequest false "name to flush; the whole cache when empty"
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/dns/cache/flush [post]
func FlushCache(w http.ResponseWriter, r *http.Request) {
	var req flushRequest
	if err := decodeBody(r, &req); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		if err := client.FlushAll(); err != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to flush the cache"))
			return
		}

		types.OK(w, "flushed")

		return
	}

	if err := client.FlushName(name); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, fmt.Sprintf("failed to flush %q", name)))
		return
	}

	types.OK(w, "flushed "+name)
}

// @Summary  Restart the DNS server (named)
// @Tags dns
// @Produce json
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/dns/restart [post]
func Restart(w http.ResponseWriter, r *http.Request) {
	if err := svc.RestartService("named"); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to restart named"))
		return
	}

	types.OK(w, "restarted")
}

// @Summary  Resolve a name through the local resolver
// @Tags dns
// @Accept json
// @Produce json
// @Param request body dns.queryRequest true "name and record type to resolve"
// @Success 200 {object} types.Response[bind.QueryResult]
// @Security BearerAuth
// @Router /api/dns/query [post]
func Query(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := decodeBody(r, &req); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		types.Error(log.Logger, w, types.NewError(http.StatusBadRequest, "name is required"))
		return
	}

	cfg, s, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	configureResolverTarget(client, cfg, s)

	result, err := client.Query(req.Name, req.Type)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, fmt.Sprintf("failed to resolve %q", req.Name)))
		return
	}

	types.OK(w, result)
}
