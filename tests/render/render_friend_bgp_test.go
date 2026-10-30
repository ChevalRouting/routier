package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
)

func friendBGPCfg() *config.Config {
	return &config.Config{
		Version: "1",
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN:      65001,
				RouterID: "10.0.0.1",
				Neighbors: []config.BGPNeighbor{{
					Address:   "10.0.0.2",
					RemoteASN: 65002,
					AddressFamilies: map[string]*config.BGPNeighborAF{
						"ipv4-unicast": {
							RouteMapIn:    "friends_edge-b_IN",
							PrefixListOut: "friends_edge-b_LOCAL",
						},
					},
				}},
			},
		},
	}
}

func friendVars() map[string]friends.Vars {
	return map[string]friends.Vars{
		"edge-b": {
			BGPPrefixLists: map[string][]config.PrefixEntry{
				"LOCAL":     {{Seq: 5, Action: "permit", Prefix: "10.9.0.0/24"}},
				"CUSTOMERS": {{Seq: 5, Action: "permit", Prefix: "192.0.2.0/24", LE: 32}},
			},
			BGPRouteMaps: map[string][]config.RouteMapEntry{
				"IN": {{
					Seq:    10,
					Action: "permit",
					Match:  map[string]string{"ip address prefix-list": "CUSTOMERS"},
					Set:    map[string]string{"local-preference": "200"},
					Call:   "COMMON",
				}},
			},
		},
	}
}

func renderFrrWithFriends(t *testing.T, cfg *config.Config, fv map[string]friends.Vars) string {
	t.Helper()

	outs, err := render.All(cfg, render.WithFriends(fv))
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	for _, o := range outs {
		if o.Name == "frr/frr.conf" {
			return o.Content
		}
	}

	t.Fatal("frr/frr.conf not rendered")
	return ""
}

func TestFriendBGPPolicyImport(t *testing.T) {
	out := renderFrrWithFriends(t, friendBGPCfg(), friendVars())

	wants := []string{
		"ip prefix-list friends_edge-b_LOCAL seq 5 permit 10.9.0.0/24",
		"ip prefix-list friends_edge-b_CUSTOMERS seq 5 permit 192.0.2.0/24 le 32",
		"route-map friends_edge-b_IN permit 10",
		"match ip address prefix-list friends_edge-b_CUSTOMERS",
		"set local-preference 200",
		"call friends_edge-b_COMMON",
		"neighbor 10.0.0.2 route-map ROUTIER-IN-10_0_0_2-ipv4-unicast in",
		"route-map ROUTIER-IN-10_0_0_2-ipv4-unicast permit 1000\n call friends_edge-b_IN\n",
		"neighbor 10.0.0.2 prefix-list friends_edge-b_LOCAL out",
	}

	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Fatalf("expected rendered frr.conf to contain %q, got:\n%s", w, out)
		}
	}
}

func TestFriendBGPSharedMerge(t *testing.T) {
	cfg := friendBGPCfg()
	cfg.Routing.BGP.Neighbors[0].AddressFamilies["ipv4-unicast"] = &config.BGPNeighborAF{
		RouteMapIn:    "friends_BOGONS_IN",
		PrefixListOut: "friends_BOGONS",
	}

	fv := map[string]friends.Vars{
		"edge-a": {
			BGPPrefixLists: map[string][]config.PrefixEntry{
				"BOGONS": {{Seq: 5, Action: "deny", Prefix: "10.0.0.0/8", LE: 32}},
			},
			BGPRouteMaps: map[string][]config.RouteMapEntry{
				"BOGONS_IN": {{Seq: 10, Action: "deny", Match: map[string]string{"ip address prefix-list": "BOGONS"}}},
			},
			BGPSharedPrefixLists: map[string]bool{"BOGONS": true},
			BGPSharedRouteMaps:   map[string]bool{"BOGONS_IN": true},
		},
		"edge-b": {
			BGPPrefixLists: map[string][]config.PrefixEntry{
				"BOGONS": {
					{Seq: 5, Action: "permit", Prefix: "203.0.113.0/24"},
					{Seq: 20, Action: "deny", Prefix: "192.0.2.0/24", LE: 32},
				},
			},
			BGPSharedPrefixLists: map[string]bool{"BOGONS": true},
		},
	}

	out := renderFrrWithFriends(t, cfg, fv)

	wants := []string{
		"ip prefix-list friends_BOGONS seq 5 deny 10.0.0.0/8 le 32",
		"ip prefix-list friends_BOGONS seq 20 deny 192.0.2.0/24 le 32",
		"route-map friends_BOGONS_IN deny 10",
		"match ip address prefix-list friends_BOGONS",
	}

	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Fatalf("expected merged output to contain %q, got:\n%s", w, out)
		}
	}

	if strings.Contains(out, "seq 5 permit 203.0.113.0/24") {
		t.Fatalf("seq 5 collision should keep first friend (edge-a) entry, got:\n%s", out)
	}

	if strings.Contains(out, "friends_edge-a_BOGONS") || strings.Contains(out, "friends_edge-b_BOGONS") {
		t.Fatalf("shared policy must not render under a per-friend namespace, got:\n%s", out)
	}
}

func TestFriendBGPImportStructured(t *testing.T) {
	res := render.FriendBGPImport(friendVars())

	var local, customers *render.ImportedPrefixList
	for i := range res.PrefixLists {
		switch res.PrefixLists[i].Name {
		case "friends_edge-b_LOCAL":
			local = &res.PrefixLists[i]
		case "friends_edge-b_CUSTOMERS":
			customers = &res.PrefixLists[i]
		}
	}

	if local == nil || customers == nil {
		t.Fatalf("expected per-friend prefix lists, got %+v", res.PrefixLists)
	}

	if local.Shared || len(local.Sources) != 1 || local.Sources[0] != "edge-b" {
		t.Fatalf("LOCAL should be unshared from edge-b, got %+v", local)
	}

	if len(res.RouteMaps) != 1 || res.RouteMaps[0].Name != "friends_edge-b_IN" {
		t.Fatalf("expected one route map friends_edge-b_IN, got %+v", res.RouteMaps)
	}

	rm := res.RouteMaps[0].Entries[0]
	if rm.Match["ip address prefix-list"] != "friends_edge-b_CUSTOMERS" {
		t.Fatalf("route-map match ref should resolve to per-friend name, got %q", rm.Match["ip address prefix-list"])
	}

	if rm.Call != "friends_edge-b_COMMON" {
		t.Fatalf("route-map call should resolve to per-friend name, got %q", rm.Call)
	}
}

func TestFriendBGPImportSharedMerged(t *testing.T) {
	fv := map[string]friends.Vars{
		"edge-a": {
			BGPPrefixLists:       map[string][]config.PrefixEntry{"BOGONS": {{Seq: 5, Action: "deny", Prefix: "10.0.0.0/8"}}},
			BGPSharedPrefixLists: map[string]bool{"BOGONS": true},
		},
		"edge-b": {
			BGPPrefixLists:       map[string][]config.PrefixEntry{"BOGONS": {{Seq: 20, Action: "deny", Prefix: "192.0.2.0/24"}}},
			BGPSharedPrefixLists: map[string]bool{"BOGONS": true},
		},
	}

	res := render.FriendBGPImport(fv)
	if len(res.PrefixLists) != 1 {
		t.Fatalf("expected a single merged prefix list, got %+v", res.PrefixLists)
	}

	pl := res.PrefixLists[0]
	if pl.Name != "friends_BOGONS" || !pl.Shared {
		t.Fatalf("expected merged shared friends_BOGONS, got %+v", pl)
	}

	if len(pl.Sources) != 2 || len(pl.Entries) != 2 {
		t.Fatalf("merged list should carry both sources and both seqs, got sources=%v entries=%v", pl.Sources, pl.Entries)
	}
}

func TestFriendBGPNoGlueAfterLocalRouteMaps(t *testing.T) {
	cfg := friendBGPCfg()
	cfg.Routing.BGP.PrefixLists = map[string][]config.PrefixEntry{
		"LOCALPL": {{Seq: 10, Action: "permit", Prefix: "10.0.0.0/8"}},
	}
	cfg.Routing.BGP.RouteMaps = map[string][]config.RouteMapEntry{
		"LOCALRM": {{Seq: 10, Action: "permit", Match: map[string]string{"ip address prefix-list": "LOCALPL"}}},
	}

	out := renderFrrWithFriends(t, cfg, friendVars())
	if strings.Contains(out, "exit!") {
		t.Fatalf("friend policies must not glue onto a local route-map's exit, got:\n%s", out)
	}

	if !strings.Contains(out, "exit\n!\nip prefix-list friends_edge-b_") && !strings.Contains(out, "exit\n\nip prefix-list friends_edge-b_") {
		t.Fatalf("expected a clean separator before imported policies, got:\n%s", out)
	}
}

func TestFriendBGPPolicyAbsentWithoutFriends(t *testing.T) {
	out := renderFrr(t, friendBGPCfg())
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "ip prefix-list friends_edge-b_") || strings.HasPrefix(line, "route-map friends_edge-b_") {
			t.Fatalf("no friend policy definitions should render without friend data, got line %q in:\n%s", line, out)
		}
	}
}
