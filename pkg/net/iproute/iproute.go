package iproute

import (
	"bytes"
	"encoding/json"
	"os/exec"
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

type Route struct {
	Dst      string
	Gateway  string
	Dev      string
	Protocol string
	Metric   int
	Family   string
}

func ShowRoutes() []Route {
	var routes []Route
	add := func(r Route) { routes = append(routes, r) }
	_ = StreamRoutes(false, add)
	_ = StreamRoutes(true, add)
	return routes
}

type RawRoute struct {
	Dst      string  `json:"dst"`
	Gateway  string  `json:"gateway"`
	Dev      string  `json:"dev"`
	Protocol string  `json:"protocol"`
	Metric   float64 `json:"metric"`
}

func StreamRoutes(ipv6 bool, fn func(Route)) error {
	args := []string{"ip", "-j", "route", "show"}
	family := "ipv4"
	if ipv6 {
		args = []string{"ip", "-j", "-6", "route", "show"}
		family = "ipv6"
	}

	raw, err := runCommand(args[0], args[1:]...)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return err
	}

	for dec.More() {
		var r RawRoute
		if err := dec.Decode(&r); err != nil {
			return err
		}

		proto := r.Protocol
		if proto == "100" {
			proto = "routier"
		}

		fn(Route{
			Dst:      r.Dst,
			Gateway:  r.Gateway,
			Dev:      r.Dev,
			Protocol: proto,
			Metric:   int(r.Metric),
			Family:   family,
		})
	}

	return nil
}

type Neighbor struct {
	Dst    string
	Dev    string
	LLAddr string
	State  string
	Family string
}

func ShowNeighbors() []Neighbor {
	var out []Neighbor
	out = append(out, showNeighborFamily(false)...)
	out = append(out, showNeighborFamily(true)...)
	return out
}

type neighborEntry struct {
	Dst    string   `json:"dst"`
	Dev    string   `json:"dev"`
	LLAddr string   `json:"lladdr"`
	State  []string `json:"state"`
}

func showNeighborFamily(ipv6 bool) []Neighbor {
	family := "ipv4"
	args := []string{"ip", "-j", "-4", "neighbor", "show"}
	if ipv6 {
		args = []string{"ip", "-j", "-6", "neighbor", "show"}
		family = "ipv6"
	}

	raw, err := runCommand(args[0], args[1:]...)
	if err != nil || len(raw) == 0 {
		return nil
	}

	var entries []neighborEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil
	}

	neighbors := make([]Neighbor, 0, len(entries))
	for _, e := range entries {
		state := "UNKNOWN"
		if len(e.State) > 0 {
			state = strings.ToUpper(e.State[0])
		}

		neighbors = append(neighbors, Neighbor{
			Dst:    e.Dst,
			Dev:    e.Dev,
			LLAddr: e.LLAddr,
			State:  state,
			Family: family,
		})
	}

	return neighbors
}
