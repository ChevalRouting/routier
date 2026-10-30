package lldp

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
)

var runCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func SetCommandRunner(fn func(name string, args ...string) ([]byte, error)) func() {
	prev := runCommand
	runCommand = fn
	return func() { runCommand = prev }
}

type Neighbor struct {
	LocalIface   string
	Protocol     string
	ChassisID    string
	ChassisName  string
	SysDescr     string
	MgmtIP       string
	PortID       string
	PortDescr    string
	Capabilities string
	VLAN         string
	Age          string
}

type document struct {
	LLDP documentFields `json:"lldp"`
}

type ifaceBody struct {
	Via     string          `json:"via"`
	Age     string          `json:"age"`
	Chassis json.RawMessage `json:"chassis"`
	Port    portBody        `json:"port"`
	VLAN    json.RawMessage `json:"vlan"`
}

type chassisBody struct {
	ID         idField         `json:"id"`
	Descr      string          `json:"descr"`
	MgmtIP     json.RawMessage `json:"mgmt-ip"`
	Capability json.RawMessage `json:"capability"`
}

type portBody struct {
	ID    idField `json:"id"`
	Descr string  `json:"descr"`
}

type idField struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type capability struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type vlanBody struct {
	VLANID string `json:"vlan-id"`
	Value  string `json:"value"`
}

func ShowNeighbors() []Neighbor {
	raw, err := runCommand("lldpcli", "-f", "json", "show", "neighbors")
	if err != nil || len(raw) == 0 {
		return nil
	}

	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}

	var out []Neighbor
	for _, entry := range namedEntries(doc.LLDP.Interface) {
		var iface ifaceBody
		if err := json.Unmarshal(entry.raw, &iface); err != nil {
			continue
		}

		out = append(out, neighborFrom(entry.name, iface))
	}

	return out
}

func neighborFrom(name string, iface ifaceBody) Neighbor {
	n := Neighbor{
		LocalIface: name,
		Protocol:   iface.Via,
		Age:        iface.Age,
		PortID:     iface.Port.ID.Value,
		PortDescr:  iface.Port.Descr,
		VLAN:       parseVLAN(iface.VLAN),
	}

	for _, entry := range namedEntries(iface.Chassis) {
		var chassis chassisBody
		if err := json.Unmarshal(entry.raw, &chassis); err != nil {
			continue
		}

		n.ChassisName = entry.name
		n.ChassisID = chassis.ID.Value
		n.SysDescr = chassis.Descr
		n.MgmtIP = parseMgmtIP(chassis.MgmtIP)
		n.Capabilities = parseCapabilities(chassis.Capability)
		break
	}

	return n
}

type namedEntry struct {
	name string
	raw  json.RawMessage
}

func namedEntries(raw json.RawMessage) []namedEntry {
	if len(raw) == 0 {
		return nil
	}

	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		var out []namedEntry
		for _, m := range arr {
			out = append(out, sortedEntries(m)...)
		}

		return out
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}

	return sortedEntries(obj)
}

func sortedEntries(m map[string]json.RawMessage) []namedEntry {
	out := make([]namedEntry, 0, len(m))
	for name, raw := range m {
		out = append(out, namedEntry{name: name, raw: raw})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func parseMgmtIP(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return one
	}

	var many []string
	if err := json.Unmarshal(raw, &many); err == nil && len(many) > 0 {
		return many[0]
	}

	return ""
}

func parseCapabilities(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	caps := unmarshalCapabilities(raw)
	enabled := make([]string, 0, len(caps))
	for _, c := range caps {
		if c.Enabled {
			enabled = append(enabled, c.Type)
		}
	}

	return strings.Join(enabled, ", ")
}

func unmarshalCapabilities(raw json.RawMessage) []capability {
	var many []capability
	if err := json.Unmarshal(raw, &many); err == nil {
		return many
	}

	var one capability
	if err := json.Unmarshal(raw, &one); err == nil {
		return []capability{one}
	}

	return nil
}

func parseVLAN(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var one vlanBody
	if err := json.Unmarshal(raw, &one); err == nil && one.VLANID != "" {
		return one.VLANID
	}

	var many []vlanBody
	if err := json.Unmarshal(raw, &many); err == nil && len(many) > 0 {
		return many[0].VLANID
	}

	return ""
}

type documentFields struct {
	Interface json.RawMessage `json:"interface"`
}
