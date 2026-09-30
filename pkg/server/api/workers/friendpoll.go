package workers

import (
	"math/rand"
	"time"

	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
)

const friendPollInterval = 10 * time.Second

func StartFriendPoll(configPath string) {
	go func() {
		for {
			friendcache.Poll(configPath)
			time.Sleep(friendPollInterval + time.Duration(rand.Int63n(int64(3*time.Second))))
		}
	}()
}
