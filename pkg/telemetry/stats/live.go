package stats

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var sysClassNet = "/sys/class/net"

type IfaceCounters struct {
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

type LiveSample struct {
	TS        int64   `json:"ts"`
	RxBps     float64 `json:"rx_bytes_ps"`
	TxBps     float64 `json:"tx_bytes_ps"`
	RxPps     float64 `json:"rx_pps"`
	TxPps     float64 `json:"tx_pps"`
	RxBytes   uint64  `json:"rx_bytes"`
	TxBytes   uint64  `json:"tx_bytes"`
	OperState string  `json:"operstate"`
}

func ReadIfaceCounters(name string) (IfaceCounters, error) {
	dir := filepath.Join(sysClassNet, name, "statistics")
	var out IfaceCounters
	for _, field := range []counterField{
		{"rx_bytes", &out.RxBytes},
		{"tx_bytes", &out.TxBytes},
		{"rx_packets", &out.RxPackets},
		{"tx_packets", &out.TxPackets},
	} {
		data, err := os.ReadFile(filepath.Join(dir, field.file))
		if err != nil {
			return IfaceCounters{}, err
		}

		value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			return IfaceCounters{}, err
		}

		*field.value = value
	}

	return out, nil
}

func ReadOperState(name string) string {
	data, err := os.ReadFile(filepath.Join(sysClassNet, name, "operstate"))
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(data))
}

func NewLiveSample(prev, cur IfaceCounters, elapsed time.Duration, at time.Time) LiveSample {
	seconds := elapsed.Seconds()
	return LiveSample{
		TS:      at.UnixMilli(),
		RxBps:   counterRate(prev.RxBytes, cur.RxBytes, seconds),
		TxBps:   counterRate(prev.TxBytes, cur.TxBytes, seconds),
		RxPps:   counterRate(prev.RxPackets, cur.RxPackets, seconds),
		TxPps:   counterRate(prev.TxPackets, cur.TxPackets, seconds),
		RxBytes: cur.RxBytes,
		TxBytes: cur.TxBytes,
	}
}

type counterField struct {
	file  string
	value *uint64
}

func counterRate(prev, cur uint64, seconds float64) float64 {
	if seconds <= 0 || cur < prev {
		return 0
	}

	return float64(cur-prev) / seconds
}
