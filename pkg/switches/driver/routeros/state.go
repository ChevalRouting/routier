package routeros

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/switches/driver"
	"github.com/ChevalRouting/routier/pkg/types"
)

const Name = "routeros-rest"

const (
	frameTypesAll        = "admit-all"
	frameTypesTaggedOnly = "admit-only-vlan-tagged"
	frameTypesAccess     = "admit-only-untagged-and-priority-tagged"
)

type identityRecord struct {
	Name string `json:"name"`
}

type resourceRecord struct {
	BoardName string `json:"board-name"`
	Version   string `json:"version"`
}

type routerboardRecord struct {
	Model  string `json:"model"`
	Serial string `json:"serial-number"`
}

type interfaceRecord struct {
	ID          string `json:".id"`
	Name        string `json:"name"`
	DefaultName string `json:"default-name"`
	Type        string `json:"type"`
	Running     string `json:"running"`
	Disabled    string `json:"disabled"`
	MAC         string `json:"mac-address"`
	Comment     string `json:"comment"`
}

type bridgeRecord struct {
	ID            string `json:".id"`
	Name          string `json:"name"`
	VLANFiltering string `json:"vlan-filtering"`
}

type bridgePortRecord struct {
	ID         string `json:".id"`
	Interface  string `json:"interface"`
	Bridge     string `json:"bridge"`
	PVID       string `json:"pvid"`
	FrameTypes string `json:"frame-types"`
	Dynamic    string `json:"dynamic"`
}

type bridgeVLANRecord struct {
	ID              string `json:".id"`
	Bridge          string `json:"bridge"`
	VLANIDs         string `json:"vlan-ids"`
	Tagged          string `json:"tagged"`
	Untagged        string `json:"untagged"`
	CurrentTagged   string `json:"current-tagged"`
	CurrentUntagged string `json:"current-untagged"`
	Dynamic         string `json:"dynamic"`
}

type bondingRecord struct {
	ID     string `json:".id"`
	Name   string `json:"name"`
	Slaves string `json:"slaves"`
	Mode   string `json:"mode"`
}

type monitorRecord struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Rate   string `json:"rate"`
}

type monitorRequest struct {
	Numbers string `json:"numbers"`
	Once    string `json:"once"`
}

type snapshot struct {
	identity    identityRecord
	resource    resourceRecord
	routerboard routerboardRecord
	interfaces  []interfaceRecord
	bridges     []bridgeRecord
	ports       []bridgePortRecord
	vlans       []bridgeVLANRecord
	bonds       []bondingRecord
	rates       map[string]string
	names       portNames
}

type Driver struct {
	client *Client
}

type readTarget struct {
	path string
	out  any
}

func New(target driver.Target) (driver.Driver, error) {
	username := target.Credentials["username"]
	if username == "" {
		return nil, errors.New("routeros: username is required")
	}

	port := "443"
	if target.Scheme == "http" {
		port = "80"
	}

	if target.Port > 0 {
		port = strconv.Itoa(target.Port)
	}

	client := NewClient(ClientConfig{
		Host: target.Host, Scheme: target.Scheme, Port: port, Username: username, Password: target.Credentials["password"],
		TLSSkipVerify: target.TLSSkipVerify,
	})
	return &Driver{client: client}, nil
}

func (d *Driver) read(ctx context.Context) (snapshot, error) {
	var s snapshot
	reads := []readTarget{
		{"/system/identity", &s.identity},
		{"/system/resource", &s.resource},
		{"/system/routerboard", &s.routerboard},
		{"/interface", &s.interfaces},
		{"/interface/bridge", &s.bridges},
		{"/interface/bridge/port", &s.ports},
		{"/interface/bridge/vlan", &s.vlans},
		{"/interface/bonding", &s.bonds},
	}
	for _, r := range reads {
		if err := d.client.Get(ctx, r.path, r.out); err != nil {
			return snapshot{}, err
		}
	}

	s.names = namesOf(s.interfaces)
	s.rates = d.rates(ctx, s.interfaces, s.names)
	s.names.normalize(&s)
	return s, nil
}

func (d *Driver) rates(ctx context.Context, interfaces []interfaceRecord, aliases portNames) map[string]string {
	var names []string
	for _, iface := range interfaces {
		if iface.Type == "ether" && iface.Running == "true" {
			names = append(names, iface.Name)
		}
	}

	rates := make(map[string]string)
	if len(names) == 0 {
		return rates
	}

	var monitored []monitorRecord
	if err := d.client.Post(ctx, "/interface/ethernet/monitor", monitorRequest{Numbers: strings.Join(names, ","), Once: ""}, &monitored); err != nil {
		return rates
	}

	for _, record := range monitored {
		rates[aliases.toIdentity(record.Name)] = record.Rate
	}

	return rates
}

func (d *Driver) State(ctx context.Context) (types.SwitchState, error) {
	s, err := d.read(ctx)
	if err != nil {
		return types.SwitchState{}, err
	}

	return buildState(s), nil
}

func buildState(s snapshot) types.SwitchState {
	state := types.SwitchState{
		Facts: types.SwitchFacts{
			Identity: s.identity.Name, Model: firstNonEmpty(s.routerboard.Model, s.resource.BoardName),
			Serial: s.routerboard.Serial, Version: s.resource.Version,
		},
		Bridges: []string{}, Ports: []types.SwitchPort{}, Bonds: []types.SwitchBond{}, VLANs: []types.SwitchVLAN{},
	}
	for _, bridge := range s.bridges {
		state.Bridges = append(state.Bridges, bridge.Name)
		state.VLANFiltering = state.VLANFiltering || bridge.VLANFiltering == "true"
	}

	bondOf := make(map[string]string)
	for _, bond := range s.bonds {
		members := splitList(bond.Slaves)
		state.Bonds = append(state.Bonds, types.SwitchBond{Name: bond.Name, Members: members, Mode: bond.Mode})
		for _, member := range members {
			bondOf[member] = bond.Name
		}
	}

	state.VLANs = vlans(s.vlans)
	bridgePorts := make(map[string]bridgePortRecord)
	for _, port := range s.ports {
		bridgePorts[port.Interface] = port
	}

	for _, iface := range s.interfaces {
		if iface.Type != "ether" && iface.Type != "bond" {
			continue
		}

		port := buildPort(iface, bridgePorts, bondOf, state.VLANs, s.rates)
		port.Label = s.names.label(port.Name)
		state.Ports = append(state.Ports, port)
	}

	return state
}

func buildPort(iface interfaceRecord, bridgePorts map[string]bridgePortRecord, bondOf map[string]string, vlans []types.SwitchVLAN, rates map[string]string) types.SwitchPort {
	port := types.SwitchPort{
		Name: iface.Name, Type: iface.Type, Running: iface.Running == "true", Disabled: iface.Disabled == "true",
		MAC: iface.MAC, Rate: rates[iface.Name], Comment: iface.Comment, Untagged: []int{}, Tagged: []int{},
	}
	if bond, ok := bondOf[iface.Name]; ok {
		port.Mode, port.Bond = types.PortBondSlave, bond
		return port
	}

	bridgePort, ok := bridgePorts[iface.Name]
	if !ok {
		port.Mode = types.PortUnbridged
		return port
	}

	port.Bridge = bridgePort.Bridge
	port.PVID, _ = strconv.Atoi(bridgePort.PVID)
	for _, vlan := range vlans {
		if vlan.Bridge != bridgePort.Bridge {
			continue
		}

		if slices.Contains(vlan.Tagged, iface.Name) {
			port.Tagged = append(port.Tagged, vlan.ID)
		}

		if slices.Contains(vlan.Untagged, iface.Name) {
			port.Untagged = append(port.Untagged, vlan.ID)
		}
	}

	port.Mode = types.PortAccess
	if len(port.Tagged) > 0 || bridgePort.FrameTypes == frameTypesTaggedOnly {
		port.Mode = types.PortTrunk
	}

	return port
}

func vlans(records []bridgeVLANRecord) []types.SwitchVLAN {
	byKey := make(map[string]*types.SwitchVLAN)
	var keys []string
	for _, record := range records {
		tagged := union(splitList(record.Tagged), splitList(record.CurrentTagged))
		untagged := union(splitList(record.Untagged), splitList(record.CurrentUntagged))
		for _, id := range expandIDs(record.VLANIDs) {
			key := record.Bridge + "/" + strconv.Itoa(id)
			vlan, ok := byKey[key]
			if !ok {
				vlan = &types.SwitchVLAN{ID: id, Bridge: record.Bridge, Tagged: []string{}, Untagged: []string{}, Dynamic: record.Dynamic == "true"}
				byKey[key] = vlan
				keys = append(keys, key)
			}

			vlan.Tagged = union(vlan.Tagged, tagged)
			vlan.Untagged = union(vlan.Untagged, untagged)
			vlan.Dynamic = vlan.Dynamic && record.Dynamic == "true"
		}
	}

	result := make([]types.SwitchVLAN, 0, len(keys))
	for _, key := range keys {
		result = append(result, *byKey[key])
	}

	slices.SortFunc(result, compareVLANs)
	return result
}

func compareVLANs(a, b types.SwitchVLAN) int {
	if a.ID != b.ID {
		return a.ID - b.ID
	}

	return strings.Compare(a.Bridge, b.Bridge)
}

func expandIDs(spec string) []int {
	var ids []int
	for _, item := range splitList(spec) {
		first, last, isRange := strings.Cut(item, "-")
		low, err := strconv.Atoi(first)
		if err != nil {
			continue
		}

		high := low
		if isRange {
			if high, err = strconv.Atoi(last); err != nil || high < low {
				continue
			}
		}

		for id := low; id <= high; id++ {
			ids = append(ids, id)
		}
	}

	return ids
}

func splitList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, item := range b {
		if !slices.Contains(out, item) {
			out = append(out, item)
		}
	}

	if out == nil {
		return []string{}
	}

	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
