package workers

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/cibi"
	webdb "github.com/ChevalRouting/routier/pkg/db"
)

func StartCibiJoin(ctx context.Context, g *Group, db *webdb.DB, self *identity.Identity, path string) {
	g.Go(func() { cibi.JoinPending(ctx, db, self, path) })
}
