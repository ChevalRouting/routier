package friendcache

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
)

var (
	friendCacheMu     sync.RWMutex
	friendIfaceCache  = map[string][]types.FriendInterface{}
	friendVarsCache   = map[string][]types.FriendVar{}
	friendConfigCache = map[string]*config.Config{}
	friendCachePath   string
)

type friendCacheFile struct {
	Status     map[string]*types.FriendStatus     `json:"status"`
	Interfaces map[string][]types.FriendInterface `json:"interfaces"`
	Vars       map[string][]types.FriendVar       `json:"vars"`
	Configs    map[string]*config.Config          `json:"configs"`
}

func SetPath(path string) {
	friendCachePath = path
}

func Load() {
	if friendCachePath == "" {
		return
	}

	data, err := os.ReadFile(friendCachePath)
	if err != nil {
		return
	}

	var f friendCacheFile
	if err := json.Unmarshal(data, &f); err != nil {
		return
	}

	friendStatusMu.Lock()
	for name, st := range f.Status {
		friendStatus[name] = st
	}

	friendStatusMu.Unlock()

	friendCacheMu.Lock()
	for name, ifaces := range f.Interfaces {
		friendIfaceCache[name] = ifaces
	}

	for name, vars := range f.Vars {
		friendVarsCache[name] = vars
	}

	for name, cfg := range f.Configs {
		friendConfigCache[name] = cfg
	}

	friendCacheMu.Unlock()
}

func saveFriendCache() {
	if friendCachePath == "" {
		return
	}

	f := friendCacheFile{
		Status:     map[string]*types.FriendStatus{},
		Interfaces: map[string][]types.FriendInterface{},
		Vars:       map[string][]types.FriendVar{},
		Configs:    map[string]*config.Config{},
	}

	friendStatusMu.RLock()
	for name, st := range friendStatus {
		f.Status[name] = st
	}

	friendStatusMu.RUnlock()

	friendCacheMu.RLock()
	for name, ifaces := range friendIfaceCache {
		f.Interfaces[name] = ifaces
	}

	for name, vars := range friendVarsCache {
		f.Vars[name] = vars
	}

	for name, cfg := range friendConfigCache {
		f.Configs[name] = cfg
	}

	friendCacheMu.RUnlock()

	if data, err := json.Marshal(f); err == nil {
		_ = os.WriteFile(friendCachePath, data, 0600)
	}
}

func CacheInterfaces(name string, ifaces []types.FriendInterface) {
	friendCacheMu.Lock()
	friendIfaceCache[name] = ifaces
	friendCacheMu.Unlock()
}

func CachedInterfaces(name string) ([]types.FriendInterface, bool) {
	friendCacheMu.RLock()
	defer friendCacheMu.RUnlock()
	ifaces, ok := friendIfaceCache[name]
	return ifaces, ok
}

func CacheVars(name string, vars []types.FriendVar) {
	friendCacheMu.Lock()
	friendVarsCache[name] = vars
	friendCacheMu.Unlock()
}

func CachedVars(name string) ([]types.FriendVar, bool) {
	friendCacheMu.RLock()
	defer friendCacheMu.RUnlock()
	vars, ok := friendVarsCache[name]
	return vars, ok
}

func CacheConfig(name string, cfg *config.Config) {
	friendCacheMu.Lock()
	friendConfigCache[name] = cfg
	friendCacheMu.Unlock()
}

func CachedConfig(name string) (*config.Config, bool) {
	friendCacheMu.RLock()
	defer friendCacheMu.RUnlock()
	cfg, ok := friendConfigCache[name]
	return cfg, ok
}

func Forget(name string) {
	friendCacheMu.Lock()
	delete(friendIfaceCache, name)
	delete(friendVarsCache, name)
	delete(friendConfigCache, name)
	friendCacheMu.Unlock()
}

func InterpolationVars() map[string]friends.Vars {
	out := map[string]friends.Vars{}

	friendStatusMu.RLock()
	for name, st := range friendStatus {
		out[name] = friends.Vars{
			Hostname:    st.Hostname,
			Fingerprint: st.Fingerprint,
			Version:     st.Version,
			Interfaces:  map[string][]string{},
		}
	}

	friendStatusMu.RUnlock()

	friendCacheMu.RLock()
	for name, ifaces := range friendIfaceCache {
		v, ok := out[name]
		if !ok {
			v = friends.Vars{Interfaces: map[string][]string{}}
		}

		if v.Interfaces == nil {
			v.Interfaces = map[string][]string{}
		}

		for _, iface := range ifaces {
			v.Interfaces[iface.Name] = iface.Addresses
		}

		out[name] = v
	}

	for name, vars := range friendVarsCache {
		v, ok := out[name]
		if !ok {
			v = friends.Vars{Interfaces: map[string][]string{}}
		}

		if v.Exports == nil {
			v.Exports = map[string]string{}
		}

		for _, ev := range vars {
			v.Exports[ev.Name] = ev.Value
		}

		out[name] = v
	}

	friendCacheMu.RUnlock()

	return out
}
