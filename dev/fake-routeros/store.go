package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type record = map[string]string

type store struct {
	mu      sync.Mutex
	objects map[string]record
	lists   map[string][]record
	nextID  int
}

var errConflict = errors.New("conflict")

func loadStore(path string) (*store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	s := &store{objects: map[string]record{}, lists: map[string][]record{}, nextID: 100}
	for path, value := range raw {
		var list []record
		if err := json.Unmarshal(value, &list); err == nil {
			s.lists[path] = list
			continue
		}

		var object record
		if err := json.Unmarshal(value, &object); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		s.objects[path] = object
	}

	return s, nil
}

func (s *store) get(path string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if object, ok := s.objects[path]; ok {
		return object, true
	}

	list, ok := s.lists[path]
	return list, ok
}

func splitItem(path string) (string, string) {
	index := strings.LastIndex(path, "/")
	return path[:index], path[index+1:]
}

func find(list []record, key, value string) int {
	return slices.IndexFunc(list, func(r record) bool { return r[key] == value })
}

func (s *store) inUse(iface string) error {
	if find(s.lists["/interface/bridge/port"], "interface", iface) >= 0 {
		return fmt.Errorf("%w: %s is a bridge port", errConflict, iface)
	}

	for _, bond := range s.lists["/interface/bonding"] {
		if slices.Contains(strings.Split(bond["slaves"], ","), iface) {
			return fmt.Errorf("%w: %s is a slave of %s", errConflict, iface, bond["name"])
		}
	}

	return nil
}

func (s *store) known(iface string) bool {
	return find(s.lists["/interface"], "name", iface) >= 0
}

func (s *store) checkMembers(body record) error {
	for _, key := range []string{"tagged", "untagged"} {
		for _, member := range strings.Split(body[key], ",") {
			if member != "" && !s.known(member) {
				return fmt.Errorf("%w: unknown interface %s", errConflict, member)
			}
		}
	}

	return nil
}

func (s *store) put(path string, body record) (record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch path {
	case "/interface/bridge/port":
		if !s.known(body["interface"]) {
			return nil, fmt.Errorf("%w: unknown interface %s", errConflict, body["interface"])
		}

		if err := s.inUse(body["interface"]); err != nil {
			return nil, err
		}
	case "/interface/bonding":
		for _, slave := range strings.Split(body["slaves"], ",") {
			if err := s.inUse(slave); err != nil {
				return nil, err
			}
		}

		s.lists["/interface"] = append(s.lists["/interface"], record{".id": s.id(), "name": body["name"], "type": "bond", "running": "true", "disabled": "false"})
	case "/interface/bridge/vlan":
		if err := s.checkMembers(body); err != nil {
			return nil, err
		}
	}

	created := record{".id": s.id(), "dynamic": "false", "disabled": "false"}
	for key, value := range body {
		created[key] = value
	}

	s.lists[path] = append(s.lists[path], created)
	return created, nil
}

func (s *store) id() string {
	s.nextID++
	return "*" + strconv.Itoa(s.nextID)
}

func (s *store) patch(path string, body record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, id := splitItem(path)
	index := find(s.lists[list], ".id", id)
	if index < 0 {
		return fmt.Errorf("no such item %s", path)
	}

	if list == "/interface/bridge/vlan" {
		if err := s.checkMembers(body); err != nil {
			return err
		}
	}

	for key, value := range body {
		s.lists[list][index][key] = value
	}

	return nil
}

func (s *store) remove(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, id := splitItem(path)
	index := find(s.lists[list], ".id", id)
	if index < 0 {
		return fmt.Errorf("no such item %s", path)
	}

	removed := s.lists[list][index]
	if list == "/interface/bonding" {
		if find(s.lists["/interface/bridge/port"], "interface", removed["name"]) >= 0 {
			return fmt.Errorf("%w: %s is still a bridge port", errConflict, removed["name"])
		}

		ifaces := s.lists["/interface"]
		s.lists["/interface"] = slices.DeleteFunc(ifaces, func(r record) bool { return r["name"] == removed["name"] })
	}

	s.lists[list] = slices.Delete(s.lists[list], index, index+1)
	return nil
}
