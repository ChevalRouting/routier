package friendcache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
)

const identityKeyPath = "/var/lib/routier/identity_ed25519"

var (
	friendStatusMu sync.RWMutex
	friendStatus   = map[string]*types.FriendStatus{}
)

type friendConn struct {
	sig    string
	client *friends.Client

	mu       sync.Mutex
	lastHash string
}

var (
	pollConnMu sync.Mutex
	pollConns  = map[string]*friendConn{}
)

func connForFriend(f *config.Friend, localID *identity.Identity) (*friendConn, error) {
	sig := f.URL + "\x00" + f.Token + "\x00" + strconv.FormatBool(f.TLSSkipVerify) + "\x00" + f.Identity.PublicKey + "\x00" + f.Identity.X25519PublicKey

	pollConnMu.Lock()
	defer pollConnMu.Unlock()
	if c, ok := pollConns[f.Name]; ok && c.sig == sig {
		return c, nil
	}

	client := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	client.SetSealOpener(localID, f.Identity.PublicKey)
	if localID != nil && f.Identity.X25519PublicKey != "" {
		key, err := localID.SharedKey(f.Identity.X25519PublicKey)
		if err != nil {
			return nil, err
		}

		client.SetSharedKey(key)
	}

	conn := &friendConn{sig: sig, client: client}
	pollConns[f.Name] = conn
	return conn, nil
}

func pruneConns(live map[string]bool) {
	pollConnMu.Lock()
	defer pollConnMu.Unlock()
	for name := range pollConns {
		if !live[name] {
			delete(pollConns, name)
		}
	}
}

func pollOnce(ctx context.Context, f *config.Friend, conn *friendConn) *types.FriendStatus {
	st := &types.FriendStatus{Name: f.Name}
	if f.Identity.X25519PublicKey == "" {
		st.LastError = "not paired for encryption"
		return st
	}

	conn.mu.Lock()
	have := conn.lastHash
	conn.mu.Unlock()

	start := time.Now()
	poll, err := conn.client.Poll(ctx, have)
	if err != nil {
		st.LastError = err.Error()
		return st
	}

	st.RTTms = time.Since(start).Milliseconds()
	st.LastSeen = time.Now().UTC().Format(time.RFC3339)
	h := poll.Hello
	st.Hostname = h.Hostname
	st.Version = h.Version
	st.OS = h.OS
	st.Fingerprint = h.IdentityFingerprint
	st.Encryption = h.Encryption
	st.ConntrackdRunning = h.ConntrackdRunning
	st.VRRP = h.VRRP

	if f.Identity.Fingerprint != "" {
		st.IdentityMatch = identity.FingerprintMatch(f.Identity.Fingerprint, h.IdentityFingerprint)
		if !st.IdentityMatch {
			st.LastError = "identity fingerprint mismatch (possible MITM)"
			return st
		}
	} else {
		st.IdentityMatch = true
	}

	st.Reachable = true

	CacheInterfaces(f.Name, poll.Interfaces)
	CacheVars(f.Name, poll.Exports)
	if len(poll.Config) > 0 {
		var rcfg config.Config
		if err := json.Unmarshal(poll.Config, &rcfg); err == nil {
			CacheConfig(f.Name, &rcfg)
		}
	}

	conn.mu.Lock()
	conn.lastHash = poll.ConfigHash
	conn.mu.Unlock()

	return st
}

func Poll(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return
	}

	localID, _ := identity.LoadOrCreate(identityKeyPath)

	live := make(map[string]bool, len(cfg.Friends))
	for _, f := range cfg.Friends {
		if !f.IsEnabled() {
			continue
		}

		live[f.Name] = true
		st := pollFriend(context.Background(), f, localID)
		friendStatusMu.Lock()
		friendStatus[f.Name] = st
		friendStatusMu.Unlock()
	}

	friendStatusMu.Lock()
	for name := range friendStatus {
		if !live[name] {
			delete(friendStatus, name)
		}
	}

	friendStatusMu.Unlock()

	friendCacheMu.Lock()
	for name := range friendIfaceCache {
		if !live[name] {
			delete(friendIfaceCache, name)
		}
	}

	for name := range friendVarsCache {
		if !live[name] {
			delete(friendVarsCache, name)
		}
	}

	for name := range friendConfigCache {
		if !live[name] {
			delete(friendConfigCache, name)
		}
	}

	friendCacheMu.Unlock()

	pruneConns(live)

	saveFriendCache()
}

func pollFriend(ctx context.Context, f *config.Friend, localID *identity.Identity) *types.FriendStatus {
	conn, err := connForFriend(f, localID)
	if err != nil {
		return &types.FriendStatus{Name: f.Name, LastError: err.Error()}
	}

	return pollOnce(ctx, f, conn)
}

func CheckOne(ctx context.Context, configPath, name string) (*types.FriendStatus, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	f := friends.Get(cfg, name)
	if f == nil {
		return nil, fmt.Errorf("friend %q not found", name)
	}

	localID, _ := identity.LoadOrCreate(identityKeyPath)
	st := pollFriend(ctx, f, localID)
	friendStatusMu.Lock()
	friendStatus[name] = st
	friendStatusMu.Unlock()

	saveFriendCache()
	return st, nil
}

type CachedState struct {
	Status     *types.FriendStatus     `json:"status,omitempty" validate:"optional"`
	Interfaces []types.FriendInterface `json:"interfaces,omitempty" validate:"optional"`
	Config     *config.Config          `json:"config,omitempty" validate:"optional"`
}

func CacheState(name string) CachedState {
	friendStatusMu.RLock()
	st := friendStatus[name]
	friendStatusMu.RUnlock()

	ifaces, _ := CachedInterfaces(name)
	if ifaces == nil {
		ifaces = []types.FriendInterface{}
	}

	cfg, _ := CachedConfig(name)
	return CachedState{Status: st, Interfaces: ifaces, Config: cfg}
}

func StatusSnapshot() map[string]types.FriendStatus {
	friendStatusMu.RLock()
	defer friendStatusMu.RUnlock()
	out := make(map[string]types.FriendStatus, len(friendStatus))
	for k, v := range friendStatus {
		out[k] = *v
	}

	return out
}
