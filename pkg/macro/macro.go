package macro

import (
	"encoding/json"

	"github.com/ChevalRouting/routier/pkg/config"
)

type Conflict struct {
	Section string
	Key     string
	Message string
}

var allSections = []string{
	"hostname", "interfaces", "tunnels", "routing", "wireguard",
	"nftables", "sysctl", "dns", "users", "services", "logging",
	"ha", "ssh", "vrfs", "gai",
	"monitoring", "boot_modules", "friends", "dhcp",
}

var mapSections = map[string]bool{
	"interfaces": true,
	"tunnels":    true,
	"wireguard":  true,
	"users":      true,
	"services":   true,
	"vrfs":       true,
	"sysctl":     true,
}

func ComputeSections(base, mod *config.Config) []string {
	var out []string
	for _, s := range allSections {
		if sectionJSON(base, s) != sectionJSON(mod, s) {
			out = append(out, s)
		}
	}

	return out
}

func sectionJSON(cfg *config.Config, section string) string {
	var v any
	switch section {
	case "hostname":
		v = cfg.Hostname
	case "interfaces":
		v = cfg.Interfaces
	case "tunnels":
		v = cfg.Tunnels
	case "routing":
		v = cfg.Routing
	case "wireguard":
		v = cfg.Wireguard
	case "nftables":
		v = cfg.Nftables
	case "sysctl":
		v = cfg.Sysctl
	case "dns":
		v = cfg.DNS
	case "users":
		v = cfg.Users
	case "services":
		v = cfg.Services
	case "logging":
		v = cfg.Logging
	case "ha":
		v = cfg.HA
	case "ssh":
		v = cfg.SSH
	case "vrfs":
		v = cfg.VRFs
	case "gai":
		v = cfg.GAI
	case "monitoring":
		v = cfg.Monitoring
	case "boot_modules":
		v = cfg.BootModules
	case "dhcp":
		v = cfg.DHCP
	default:
		return ""
	}

	b, _ := json.Marshal(v)
	return string(b)
}

func sectionMapKeys(cfg *config.Config, section string) map[string]string {
	asMap := func(v any) map[string]string {
		b, _ := json.Marshal(v)
		var raw map[string]json.RawMessage
		if json.Unmarshal(b, &raw) != nil {
			return nil
		}

		out := make(map[string]string, len(raw))
		for k, rv := range raw {
			out[k] = string(rv)
		}

		return out
	}

	switch section {
	case "interfaces":
		return asMap(cfg.Interfaces)
	case "tunnels":
		return asMap(cfg.Tunnels)
	case "wireguard":
		return asMap(cfg.Wireguard)
	case "users":
		return asMap(cfg.Users)
	case "services":
		return asMap(cfg.Services)
	case "vrfs":
		return asMap(cfg.VRFs)
	case "sysctl":
		return asMap(cfg.Sysctl)
	}

	return nil
}

func changedKeys(base, mod map[string]string) map[string]struct{} {
	out := make(map[string]struct{})
	for k, mv := range mod {
		if bv, ok := base[k]; !ok || bv != mv {
			out[k] = struct{}{}
		}
	}

	for k := range base {
		if _, ok := mod[k]; !ok {
			out[k] = struct{}{}
		}
	}

	return out
}

func orEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}

	return m
}

func DetectConflicts(base, live, mod *config.Config, sections []string) []Conflict {
	var conflicts []Conflict
	for _, section := range sections {
		if sectionJSON(base, section) == sectionJSON(live, section) {
			continue
		}

		if mapSections[section] {
			macroChanged := changedKeys(orEmpty(sectionMapKeys(base, section)), orEmpty(sectionMapKeys(mod, section)))
			liveChanged := changedKeys(orEmpty(sectionMapKeys(base, section)), orEmpty(sectionMapKeys(live, section)))
			for key := range macroChanged {
				if _, hit := liveChanged[key]; hit {
					conflicts = append(conflicts, Conflict{
						Section: section,
						Key:     key,
						Message: "modified in both macro and current config",
					})
				}
			}
		} else {
			conflicts = append(conflicts, Conflict{
				Section: section,
				Message: "modified in current config since macro was recorded",
			})
		}
	}

	return conflicts
}

func ApplyDelta(base, live, mod *config.Config, sections []string) *config.Config {
	baseJ, _ := json.Marshal(base)
	liveJ, _ := json.Marshal(live)
	modJ, _ := json.Marshal(mod)

	var baseMap, liveMap, modMap map[string]json.RawMessage
	_ = json.Unmarshal(baseJ, &baseMap)
	_ = json.Unmarshal(liveJ, &liveMap)
	_ = json.Unmarshal(modJ, &modMap)

	for _, s := range sections {
		switch {
		case mapSections[s]:
			liveMap[s] = applyObjDeltaJSON(baseMap[s], liveMap[s], modMap[s])
		case s == "routing":
			liveMap["routing"] = applyRoutingDeltaJSON(baseMap["routing"], liveMap["routing"], modMap["routing"])
		default:
			if v, ok := modMap[s]; ok {
				liveMap[s] = v
			} else {
				delete(liveMap, s)
			}
		}
	}

	merged, _ := json.Marshal(liveMap)
	var result config.Config
	_ = json.Unmarshal(merged, &result)
	result.BaseDir = live.BaseDir
	return &result
}

func rawObjMap(r json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if r != nil {
		_ = json.Unmarshal(r, &m)
	}

	if m == nil {
		m = map[string]json.RawMessage{}
	}

	return m
}

func rawList(r json.RawMessage) []json.RawMessage {
	var items []json.RawMessage
	if r != nil {
		_ = json.Unmarshal(r, &items)
	}

	return items
}

func applyObjDeltaJSON(baseR, liveR, modR json.RawMessage) json.RawMessage {
	base := rawObjMap(baseR)
	live := rawObjMap(liveR)
	mod := rawObjMap(modR)

	for k, mv := range mod {
		if bv, ok := base[k]; !ok || string(bv) != string(mv) {
			live[k] = mv
		}
	}

	for k := range base {
		if _, ok := mod[k]; !ok {
			delete(live, k)
		}
	}

	out, _ := json.Marshal(live)
	return out
}

func applyListDeltaJSON(baseR, liveR, modR json.RawMessage, keyField string) json.RawMessage {
	itemKey := func(raw json.RawMessage) string {
		var m map[string]json.RawMessage
		if json.Unmarshal(raw, &m) != nil {
			return string(raw)
		}

		if v, ok := m[keyField]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				return s
			}

			return string(v)
		}

		return string(raw)
	}

	baseItems := rawList(baseR)
	liveItems := rawList(liveR)
	modItems := rawList(modR)

	baseByKey := make(map[string]json.RawMessage, len(baseItems))
	for _, it := range baseItems {
		baseByKey[itemKey(it)] = it
	}

	modByKey := make(map[string]json.RawMessage, len(modItems))
	for _, it := range modItems {
		modByKey[itemKey(it)] = it
	}

	liveByKey := make(map[string]json.RawMessage, len(liveItems))
	liveOrder := make([]string, 0, len(liveItems))
	for _, it := range liveItems {
		k := itemKey(it)
		if _, exists := liveByKey[k]; !exists {
			liveOrder = append(liveOrder, k)
		}

		liveByKey[k] = it
	}

	for k, mv := range modByKey {
		bv, inBase := baseByKey[k]
		if !inBase {
			if _, inLive := liveByKey[k]; !inLive {
				liveOrder = append(liveOrder, k)
			}

			liveByKey[k] = mv
		} else if string(bv) != string(mv) {
			liveByKey[k] = mv
		}
	}

	for k := range baseByKey {
		if _, inMod := modByKey[k]; !inMod {
			delete(liveByKey, k)
		}
	}

	result := make([]json.RawMessage, 0, len(liveOrder))
	for _, k := range liveOrder {
		if it, ok := liveByKey[k]; ok {
			result = append(result, it)
		}
	}

	out, _ := json.Marshal(result)
	return out
}

func applyRoutingDeltaJSON(baseR, liveR, modR json.RawMessage) json.RawMessage {
	base := rawObjMap(baseR)
	live := rawObjMap(liveR)
	mod := rawObjMap(modR)

	if string(base["static"]) != string(mod["static"]) {
		live["static"] = applyListDeltaJSON(base["static"], live["static"], mod["static"], "destination")
	}

	if string(base["bgp"]) != string(mod["bgp"]) {
		baseBGP := rawObjMap(base["bgp"])
		liveBGP := rawObjMap(live["bgp"])
		modBGP := rawObjMap(mod["bgp"])

		if string(baseBGP["neighbors"]) != string(modBGP["neighbors"]) {
			liveBGP["neighbors"] = applyListDeltaJSON(baseBGP["neighbors"], liveBGP["neighbors"], modBGP["neighbors"], "address")
		}

		for _, f := range []string{"address_families", "prefix_lists", "route_maps"} {
			if string(baseBGP[f]) != string(modBGP[f]) {
				liveBGP[f] = applyObjDeltaJSON(baseBGP[f], liveBGP[f], modBGP[f])
			}
		}

		for _, f := range []string{"asn", "router_id", "no_ebgp_requires_policy", "no_default_ipv4_unicast", "no_import_check"} {
			if string(baseBGP[f]) != string(modBGP[f]) {
				if v, ok := modBGP[f]; ok {
					liveBGP[f] = v
				} else {
					delete(liveBGP, f)
				}
			}
		}

		merged, _ := json.Marshal(liveBGP)
		live["bgp"] = merged
	}

	for _, f := range []string{"ospf", "ospf6", "anycast", "radvd", "pbr", "bfd", "vrfs"} {
		if string(base[f]) != string(mod[f]) {
			if v, ok := mod[f]; ok {
				live[f] = v
			} else {
				delete(live, f)
			}
		}
	}

	out, _ := json.Marshal(live)
	return out
}
