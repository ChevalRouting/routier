package routecache

import (
	"database/sql"
	"strconv"
	"sync"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/iproute"
	"github.com/ChevalRouting/routier/pkg/types"
)

const (
	settingRoutesSnapshotAt = "routes_snapshot_at"
	routesSnapshotTTL       = 30 * time.Second
)

var routesRefreshMu sync.Mutex

func MaybeRefresh(db *sql.DB, force bool, ttl time.Duration) error {
	routesRefreshMu.Lock()
	defer routesRefreshMu.Unlock()

	if !force && routesSnapshotFresh(db, ttl) {
		return nil
	}

	return refreshKernelRoutes(db)
}

func routesSnapshotFresh(db *sql.DB, ttl time.Duration) bool {
	if ttl <= 0 {
		ttl = routesSnapshotTTL
	}

	ts, err := strconv.ParseInt(webdb.Setting(db, settingRoutesSnapshotAt, "0"), 10, 64)
	if err != nil || ts == 0 {
		return false
	}

	return time.Since(time.Unix(ts, 0)) < ttl
}

func refreshKernelRoutes(db *sql.DB) error {
	var routes []types.KernelRoute
	collect := func(rt iproute.Route) {
		routes = append(routes, types.KernelRoute{
			Dst: rt.Dst, Gateway: rt.Gateway, Dev: rt.Dev,
			Protocol: rt.Protocol, Metric: rt.Metric, Family: rt.Family,
		})
	}

	if err := iproute.StreamRoutes(false, collect); err != nil {
		return err
	}

	if err := iproute.StreamRoutes(true, collect); err != nil {
		return err
	}

	if err := webdb.ReplaceKernelRoutes(db, routes); err != nil {
		return err
	}

	return webdb.SetSetting(db, settingRoutesSnapshotAt, strconv.FormatInt(time.Now().Unix(), 10))
}
