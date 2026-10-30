package api

import (
	"crypto/subtle"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
)

const friendTokenTTL = 2 * time.Second

type friendTokenCache struct {
	mu       sync.Mutex
	path     string
	loadedAt time.Time
	friends  []*config.Friend
}

var friendTokens = &friendTokenCache{}

func (c *friendTokenCache) lookup(path, token string, now time.Time) *config.Friend {
	for _, f := range c.snapshot(path, now) {
		if subtle.ConstantTimeCompare([]byte(f.Token), []byte(token)) == 1 {
			return f
		}
	}

	return nil
}

func (c *friendTokenCache) snapshot(path string, now time.Time) []*config.Friend {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == path && now.Sub(c.loadedAt) < friendTokenTTL {
		return c.friends
	}

	c.path, c.loadedAt, c.friends = path, now, nil

	cfg, err := config.Load(path)
	if err != nil {
		return nil
	}

	for _, f := range cfg.Friends {
		if f.IsEnabled() && f.Token != "" {
			c.friends = append(c.friends, f)
		}
	}

	return c.friends
}
