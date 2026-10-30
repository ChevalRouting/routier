package routeros

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/switches/driver"
	"github.com/ChevalRouting/routier/pkg/types"
)

const defaultBondMode = "802.3ad"

type phase int

const (
	phaseVLANShrink phase = iota
	phasePortDelete
	phaseBondDelete
	phaseBondCreate
	phasePortAdd
	phasePortPatch
	phaseVLANGrow
)

type restStep struct {
	client *Client
	phase  phase
	method string
	path   string
	body   map[string]string
	skip   bool
}

func (s restStep) Describe() string {
	keys := make([]string, 0, len(s.body))
	for key := range s.body {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	parts := []string{s.method, "/rest" + s.path}
	for _, key := range keys {
		parts = append(parts, key+"="+strconv.Quote(s.body[key]))
	}

	return strings.Join(parts, " ")
}

func (s restStep) Run(ctx context.Context) error {
	switch s.method {
	case http.MethodPatch:
		return s.client.Patch(ctx, s.path, s.body)
	case http.MethodPut:
		return s.client.Put(ctx, s.path, s.body, nil)
	case http.MethodDelete:
		return s.client.Delete(ctx, s.path)
	}

	return fmt.Errorf("unsupported method %s", s.method)
}

type vlanEntry struct {
	id           string
	bridge       string
	ids          []int
	origIDs      []int
	tagged       []string
	untagged     []string
	origTagged   []string
	origUntagged []string
}

type portSettings struct {
	bridge     string
	pvid       string
	frameTypes string
	tagged     []int
	untagged   []int
}

type planner struct {
	client  *Client
	names   portNames
	bridge  string
	entries []*vlanEntry
	ports   map[string]bridgePortRecord
	bonds   map[string]bondingRecord
	steps   []restStep
	added   map[string]int
}

func newPlanner(client *Client, s snapshot) *planner {
	p := &planner{client: client, names: s.names, bridge: defaultBridge(s.bridges), ports: make(map[string]bridgePortRecord), bonds: make(map[string]bondingRecord), added: make(map[string]int)}

	for _, port := range s.ports {
		if port.Dynamic != "true" {
			p.ports[port.Interface] = port
		}
	}

	for _, bond := range s.bonds {
		p.bonds[bond.Name] = bond
	}

	for _, record := range s.vlans {
		if record.Dynamic == "true" {
			continue
		}

		ids := expandIDs(record.VLANIDs)
		tagged, untagged := splitList(record.Tagged), splitList(record.Untagged)
		p.entries = append(p.entries, &vlanEntry{
			id: record.ID, bridge: record.Bridge, ids: ids, origIDs: slices.Clone(ids),
			tagged: tagged, untagged: untagged, origTagged: slices.Clone(tagged), origUntagged: slices.Clone(untagged),
		})
	}

	return p
}

func defaultBridge(bridges []bridgeRecord) string {
	for _, bridge := range bridges {
		if bridge.VLANFiltering == "true" {
			return bridge.Name
		}
	}

	if len(bridges) > 0 {
		return bridges[0].Name
	}

	return ""
}

func (p *planner) add(step restStep) {
	step.client = p.client
	p.steps = append(p.steps, step)
}

func (p *planner) bridgeOf(port string) string {
	if bp, ok := p.ports[port]; ok {
		return bp.Bridge
	}

	if index, ok := p.added[port]; ok {
		return p.steps[index].body["bridge"]
	}

	return p.bridge
}

func (p *planner) entryFor(bridge string, vid int) *vlanEntry {
	for _, entry := range p.entries {
		if entry.bridge != bridge || !slices.Contains(entry.ids, vid) {
			continue
		}

		if len(entry.ids) == 1 {
			return entry
		}

		entry.ids = slices.DeleteFunc(entry.ids, func(id int) bool { return id == vid })
		split := &vlanEntry{bridge: bridge, ids: []int{vid}, tagged: slices.Clone(entry.tagged), untagged: slices.Clone(entry.untagged)}
		p.entries = append(p.entries, split)
		return split
	}

	created := &vlanEntry{bridge: bridge, ids: []int{vid}}
	p.entries = append(p.entries, created)
	return created
}

func (p *planner) detach(port string) {
	for _, entry := range p.entries {
		entry.tagged = without(entry.tagged, port)
		entry.untagged = without(entry.untagged, port)
	}
}

func (p *planner) settings(port string) portSettings {
	settings := portSettings{bridge: p.bridgeOf(port), pvid: "1", frameTypes: frameTypesAll}
	if bp, ok := p.ports[port]; ok {
		settings.pvid, settings.frameTypes = bp.PVID, bp.FrameTypes
	}

	for _, entry := range p.entries {
		if slices.Contains(entry.tagged, port) {
			settings.tagged = append(settings.tagged, entry.ids...)
		}

		if slices.Contains(entry.untagged, port) {
			settings.untagged = append(settings.untagged, entry.ids...)
		}
	}

	return settings
}

func (p *planner) configurePort(port string, settings portSettings) {
	body := map[string]string{"pvid": settings.pvid, "frame-types": settings.frameTypes}
	if bp, ok := p.ports[port]; ok {
		p.add(restStep{phase: phasePortPatch, method: http.MethodPatch, path: "/interface/bridge/port/" + bp.ID, body: body})
		return
	}

	if index, ok := p.added[port]; ok {
		p.steps[index].body["pvid"], p.steps[index].body["frame-types"] = settings.pvid, settings.frameTypes
		return
	}

	body["bridge"], body["interface"] = settings.bridge, port
	p.added[port] = len(p.steps)
	p.add(restStep{phase: phasePortAdd, method: http.MethodPut, path: "/interface/bridge/port", body: body})
}

func (p *planner) join(port string, settings portSettings) {
	for _, vid := range settings.tagged {
		entry := p.entryFor(settings.bridge, vid)
		entry.tagged = with(entry.tagged, port)
	}

	for _, vid := range settings.untagged {
		entry := p.entryFor(settings.bridge, vid)
		entry.untagged = with(entry.untagged, port)
	}
}

func (p *planner) setAccess(port string, vlan int) {
	p.detach(port)
	settings := portSettings{bridge: p.bridgeOf(port), pvid: strconv.Itoa(vlan), frameTypes: frameTypesAccess, untagged: []int{vlan}}
	p.configurePort(port, settings)
	p.join(port, settings)
}

func (p *planner) setTrunk(port string, native int, tagged []int) {
	p.detach(port)
	settings := portSettings{bridge: p.bridgeOf(port), pvid: strconv.Itoa(max(native, 1)), frameTypes: frameTypesTaggedOnly}
	if native > 1 {
		settings.frameTypes = frameTypesAll
		settings.untagged = []int{native}
	}

	for _, vid := range tagged {
		if vid != native {
			settings.tagged = append(settings.tagged, vid)
		}
	}

	p.configurePort(port, settings)
	p.join(port, settings)
}

func (p *planner) createBond(name string, members []string, mode string) {
	if mode == "" {
		mode = defaultBondMode
	}

	settings := p.settings(members[0])
	for _, member := range members {
		p.detach(member)
		if index, ok := p.added[member]; ok {
			p.steps[index].skip = true
			delete(p.added, member)
		}

		if bp, ok := p.ports[member]; ok {
			p.add(restStep{phase: phasePortDelete, method: http.MethodDelete, path: "/interface/bridge/port/" + bp.ID})
			delete(p.ports, member)
		}
	}

	p.add(restStep{phase: phaseBondCreate, method: http.MethodPut, path: "/interface/bonding", body: map[string]string{
		"name": name, "slaves": strings.Join(members, ","), "mode": mode,
	}})
	p.configurePort(name, settings)
	p.join(name, settings)
}

func (p *planner) deleteBond(name string) {
	bond := p.bonds[name]
	settings := p.settings(name)
	p.detach(name)
	if bp, ok := p.ports[name]; ok {
		p.add(restStep{phase: phasePortDelete, method: http.MethodDelete, path: "/interface/bridge/port/" + bp.ID})
		delete(p.ports, name)
	}

	p.add(restStep{phase: phaseBondDelete, method: http.MethodDelete, path: "/interface/bonding/" + bond.ID})
	for _, member := range splitList(bond.Slaves) {
		p.configurePort(member, settings)
		p.join(member, settings)
	}
}

func (p *planner) apply(change types.PortChange) {
	switch change.Kind {
	case types.ChangeSetAccess:
		for _, port := range change.Ports {
			p.setAccess(port, change.VLAN)
		}
	case types.ChangeSetTrunk:
		for _, port := range change.Ports {
			p.setTrunk(port, change.VLAN, change.Tagged)
		}
	case types.ChangeCreateBond:
		p.createBond(change.Bond, change.Ports, change.Mode)
	case types.ChangeDeleteBond:
		p.deleteBond(change.Bond)
	}
}

func (p *planner) finalize() []restStep {
	for _, entry := range p.entries {
		p.finalizeEntry(entry)
	}

	steps := slices.DeleteFunc(p.steps, func(step restStep) bool { return step.skip })
	for _, step := range steps {
		p.names.deviceBody(step.body)
	}

	sort.SliceStable(steps, func(i, j int) bool { return steps[i].phase < steps[j].phase })
	return steps
}

func (p *planner) finalizeEntry(entry *vlanEntry) {
	vlanIDs := joinIDs(entry.ids)
	if entry.id == "" {
		if len(entry.tagged) == 0 && len(entry.untagged) == 0 {
			return
		}

		p.add(restStep{phase: phaseVLANGrow, method: http.MethodPut, path: "/interface/bridge/vlan", body: map[string]string{
			"bridge": entry.bridge, "vlan-ids": vlanIDs, "tagged": strings.Join(entry.tagged, ","), "untagged": strings.Join(entry.untagged, ","),
		}})
		return
	}

	path := "/interface/bridge/vlan/" + entry.id
	if len(entry.tagged) == 0 && len(entry.untagged) == 0 && (len(entry.origTagged) > 0 || len(entry.origUntagged) > 0) {
		p.add(restStep{phase: phaseVLANShrink, method: http.MethodDelete, path: path})
		return
	}

	keptTagged, keptUntagged := intersect(entry.origTagged, entry.tagged), intersect(entry.origUntagged, entry.untagged)
	if !slices.Equal(entry.ids, entry.origIDs) || !slices.Equal(keptTagged, entry.origTagged) || !slices.Equal(keptUntagged, entry.origUntagged) {
		p.add(restStep{phase: phaseVLANShrink, method: http.MethodPatch, path: path, body: map[string]string{
			"vlan-ids": vlanIDs, "tagged": strings.Join(keptTagged, ","), "untagged": strings.Join(keptUntagged, ","),
		}})
	}

	if !slices.Equal(keptTagged, entry.tagged) || !slices.Equal(keptUntagged, entry.untagged) {
		p.add(restStep{phase: phaseVLANGrow, method: http.MethodPatch, path: path, body: map[string]string{
			"tagged": strings.Join(entry.tagged, ","), "untagged": strings.Join(entry.untagged, ","),
		}})
	}
}

func (d *Driver) Plan(ctx context.Context, plan types.SwitchPlan) ([]driver.Step, error) {
	s, err := d.read(ctx)
	if err != nil {
		return nil, err
	}

	p := newPlanner(d.client, s)
	for _, change := range plan.Changes {
		p.apply(change)
	}

	planned := p.finalize()
	steps := make([]driver.Step, 0, len(planned))
	for _, step := range planned {
		steps = append(steps, step)
	}

	return steps, nil
}

func joinIDs(ids []int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}

	return strings.Join(parts, ",")
}

func with(items []string, item string) []string {
	if slices.Contains(items, item) {
		return items
	}

	return append(items, item)
}

func without(items []string, item string) []string {
	return slices.DeleteFunc(slices.Clone(items), func(candidate string) bool { return candidate == item })
}

func intersect(original, current []string) []string {
	kept := []string{}
	for _, item := range original {
		if slices.Contains(current, item) {
			kept = append(kept, item)
		}
	}

	return kept
}
