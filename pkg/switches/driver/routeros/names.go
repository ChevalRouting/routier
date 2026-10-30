package routeros

import "strings"

type portNames struct {
	identity map[string]string
	device   map[string]string
}

func namesOf(interfaces []interfaceRecord) portNames {
	names := portNames{identity: make(map[string]string), device: make(map[string]string)}
	for _, iface := range interfaces {
		if iface.DefaultName != "" && iface.DefaultName != iface.Name {
			names.identity[iface.Name] = iface.DefaultName
			names.device[iface.DefaultName] = iface.Name
		}
	}

	return names
}

func (n portNames) toIdentity(name string) string {
	if identity, ok := n.identity[name]; ok {
		return identity
	}

	return name
}

func (n portNames) toDevice(name string) string {
	if device, ok := n.device[name]; ok {
		return device
	}

	return name
}

func (n portNames) label(identity string) string {
	return n.device[identity]
}

func mapList(list string, rename func(string) string) string {
	items := splitList(list)
	for index, item := range items {
		items[index] = rename(item)
	}

	return strings.Join(items, ",")
}

func (n portNames) normalize(s *snapshot) {
	if len(n.identity) == 0 {
		return
	}

	for index := range s.interfaces {
		s.interfaces[index].Name = n.toIdentity(s.interfaces[index].Name)
	}

	for index := range s.ports {
		s.ports[index].Interface = n.toIdentity(s.ports[index].Interface)
	}

	for index := range s.bonds {
		s.bonds[index].Slaves = mapList(s.bonds[index].Slaves, n.toIdentity)
	}

	for index := range s.vlans {
		vlan := &s.vlans[index]
		vlan.Tagged, vlan.Untagged = mapList(vlan.Tagged, n.toIdentity), mapList(vlan.Untagged, n.toIdentity)
		vlan.CurrentTagged, vlan.CurrentUntagged = mapList(vlan.CurrentTagged, n.toIdentity), mapList(vlan.CurrentUntagged, n.toIdentity)
	}
}

func (n portNames) deviceBody(body map[string]string) {
	if len(n.device) == 0 {
		return
	}

	if iface, ok := body["interface"]; ok {
		body["interface"] = n.toDevice(iface)
	}

	for _, key := range []string{"slaves", "tagged", "untagged"} {
		if list, ok := body[key]; ok {
			body[key] = mapList(list, n.toDevice)
		}
	}
}
