package keepalived

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Peer struct {
	IP       string
	Priority int
	LastSeen string
}

type InstanceState struct {
	State    string
	MasterIP string
	Peers    []Peer
}

const (
	dataDir  = "/run/keepalived"
	dataFile = dataDir + "/keepalived.data"
)

var dumpMu sync.Mutex

func States() map[string]*InstanceState {
	dumpMu.Lock()
	defer dumpMu.Unlock()

	_ = os.MkdirAll(dataDir, 0o755)
	_ = os.Remove(dataFile)
	signalKeepalived()
	waitDumpWritten(500 * time.Millisecond)

	return parseData(dataFile)
}

func waitDumpWritten(timeout time.Duration) {
	lastSize := int64(-1)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		info, err := os.Stat(dataFile)
		if err != nil {
			continue
		}

		if info.Size() > 0 && info.Size() == lastSize {
			return
		}

		lastSize = info.Size()
	}
}

func signalKeepalived() {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		raw, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err != nil {
			continue
		}

		args := strings.Split(string(raw), "\x00")
		if len(args) == 0 || !strings.HasSuffix(args[0], "keepalived") {
			continue
		}

		if pid, err := strconv.Atoi(e.Name()); err == nil {
			_ = syscall.Kill(pid, syscall.SIGUSR1)
			return
		}
	}
}

func parseData(path string) map[string]*InstanceState {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}

	defer func(action func() error) { _ = action() }(f.Close)

	return ParseStates(f)
}

func ParseStates(r io.Reader) map[string]*InstanceState {
	states := make(map[string]*InstanceState)
	var current string
	var inPeers bool

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		if line == "" {
			inPeers = false
			continue
		}

		if line == "Peers:" {
			inPeers = true
			continue
		}

		if inPeers && current != "" {
			parsePeerLine(line, current, states)
			continue
		}

		if strings.HasPrefix(line, "---") {
			current = ""
			inPeers = false
			continue
		}

		before, after, ok := strings.Cut(line, " = ")
		if !ok {
			continue
		}

		key := strings.TrimSpace(before)
		val := strings.TrimSpace(after)
		switch key {
		case "VRRP Instance":
			current = val
			inPeers = false
			if _, exists := states[current]; !exists {
				states[current] = &InstanceState{}
			}
		case "State":
			if current != "" {
				states[current].State = val
			}
		case "Master router":
			if current != "" {
				states[current].MasterIP = val
			}
		}
	}

	_ = scanner.Err()

	return states
}

func parsePeerLine(line, instance string, states map[string]*InstanceState) {
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return
	}

	peer := Peer{IP: fields[0]}
	if _, after, ok := strings.Cut(line, "at priority "); ok {
		rest := strings.TrimSpace(after)
		if b, _, ok := strings.Cut(rest, " "); ok {
			rest = b
		}

		if p, err := strconv.Atoi(rest); err == nil {
			peer.Priority = p
		}
	}

	if _, lastSeen, ok := strings.Cut(line, "received "); ok {
		af, _, ok := strings.Cut(lastSeen, " at priority")
		if ok {
			lastSeen = strings.TrimSpace(af)
		}

		peer.LastSeen = lastSeen
	}

	states[instance].Peers = append(states[instance].Peers, peer)
}
