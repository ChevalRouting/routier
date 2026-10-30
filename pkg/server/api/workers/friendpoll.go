package workers

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
)

const friendPollInterval = 10 * time.Second

func StartFriendPoll(ctx context.Context, g *Group, configPath string) {
	g.Go(func() { every(ctx, friendPollJitter, func() { friendcache.Poll(configPath) }) })
}

func friendPollJitter() time.Duration {
	return friendPollInterval + rand.N(3*time.Second)
}
