package api

import (
	"crypto/subtle"
	"slices"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
)

const (
	friendTokenTTL = 2 * time.Second
	builtinAdmin   = "routier"
)

type friendTokenCache struct {
	mu       sync.Mutex
	path     string
	loadedAt time.Time
	friends  []*config.Friend
	admins   map[string]bool
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

func (c *friendTokenCache) adminAllowed(path, username string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.refresh(path, now)

	return c.admins == nil || c.admins[username]
}

func (c *friendTokenCache) snapshot(path string, now time.Time) []*config.Friend {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.refresh(path, now)

	return c.friends
}

func (c *friendTokenCache) refresh(path string, now time.Time) {
	if c.path == path && now.Sub(c.loadedAt) < friendTokenTTL {
		return
	}

	c.path, c.loadedAt, c.friends, c.admins = path, now, nil, nil

	cfg, err := config.Load(path)
	if err != nil {
		return
	}

	c.admins = map[string]bool{builtinAdmin: true}
	for name, u := range cfg.Users {
		if name != "root" && u != nil && slices.Contains(u.Groups, "wheel") {
			c.admins[name] = true
		}
	}

	for _, f := range cfg.Friends {
		if f.IsEnabled() && f.Token != "" {
			c.friends = append(c.friends, f)
		}
	}
}
