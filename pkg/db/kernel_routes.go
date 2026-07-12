package db

import (
	"database/sql"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureKernelRoutesSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS kernel_routes (
		dst      TEXT    NOT NULL DEFAULT '',
		gateway  TEXT    NOT NULL DEFAULT '',
		dev      TEXT    NOT NULL DEFAULT '',
		protocol TEXT    NOT NULL DEFAULT '',
		metric   INTEGER NOT NULL DEFAULT 0,
		family   TEXT    NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_kernel_routes ON kernel_routes(family, dst)`)

	return nil
}

type Filter struct {
	Query       string
	Family      string
	Protocols   []string
	DefaultOnly bool
}

func (f Filter) where() (string, []any) {
	var conds []string
	var args []any

	if f.DefaultOnly {
		conds = append(conds, "dst IN ('default', '0.0.0.0/0', '::/0')")
	}

	if f.Family != "" {
		conds = append(conds, "family = ?")
		args = append(args, f.Family)
	}

	if len(f.Protocols) > 0 {
		var or []string
		for _, kw := range f.Protocols {
			or = append(or, "LOWER(protocol) LIKE ?")
			args = append(args, "%"+strings.ToLower(kw)+"%")
		}

		conds = append(conds, "("+strings.Join(or, " OR ")+")")
	}

	if f.Query != "" {
		like := "%" + strings.ToLower(f.Query) + "%"
		conds = append(conds, "(LOWER(dst) LIKE ? OR LOWER(gateway) LIKE ? OR LOWER(dev) LIKE ? OR LOWER(protocol) LIKE ?)")
		args = append(args, like, like, like, like)
	}

	if len(conds) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func ReplaceKernelRoutes(db *sql.DB, routes []types.KernelRoute) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM kernel_routes"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO kernel_routes(dst, gateway, dev, protocol, metric, family) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, rt := range routes {
		if _, err := stmt.Exec(rt.Dst, rt.Gateway, rt.Dev, rt.Protocol, rt.Metric, rt.Family); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func QueryKernelRoutes(db *sql.DB, f Filter, offset, limit int) ([]types.KernelRoute, int, error) {
	where, args := f.where()

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM kernel_routes"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sqlStr := "SELECT dst, gateway, dev, protocol, metric, family FROM kernel_routes" + where + " ORDER BY family, dst"
	qArgs := args
	if !f.DefaultOnly {
		sqlStr += " LIMIT ? OFFSET ?"
		qArgs = append(append([]any{}, args...), limit, offset)
	}

	rows, err := db.Query(sqlStr, qArgs...)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	routes := []types.KernelRoute{}
	for rows.Next() {
		var rt types.KernelRoute
		if err := rows.Scan(&rt.Dst, &rt.Gateway, &rt.Dev, &rt.Protocol, &rt.Metric, &rt.Family); err != nil {
			return nil, 0, err
		}

		routes = append(routes, rt)
	}

	return routes, total, rows.Err()
}
