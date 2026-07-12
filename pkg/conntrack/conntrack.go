package conntrack

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

type Breakdown struct {
	ByProto   map[string]int64
	TCPStates map[string]int64
}

var runCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func SetCommandRunner(fn func(name string, args ...string) ([]byte, error)) func() {
	prev := runCommand
	runCommand = fn
	return func() { runCommand = prev }
}

func Count() (int64, error) {
	out, err := runCommand("conntrack", "-C")
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

func List() (*Breakdown, error) {
	out, err := runCommand("conntrack", "-L")
	if err != nil {
		return nil, err
	}

	return parseList(out), nil
}

func parseList(out []byte) *Breakdown {
	byProto := make(map[string]int64)
	tcpStates := make(map[string]int64)

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		proto := fields[0]
		if strings.ContainsAny(proto, "=:") {
			continue
		}

		byProto[proto]++
		if proto == "tcp" && len(fields) >= 4 && !strings.Contains(fields[3], "=") {
			tcpStates[fields[3]]++
		}
	}

	if len(byProto) == 0 {
		byProto = nil
	}

	if len(tcpStates) == 0 {
		tcpStates = nil
	}

	return &Breakdown{ByProto: byProto, TCPStates: tcpStates}
}
