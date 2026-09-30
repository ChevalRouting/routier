package probe

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var summaryPattern = regexp.MustCompile(`(?:=|:)\s*[0-9.]+/([0-9.]+)/[0-9.]+(?:/[0-9.]+)?\s*ms`)

func Ping(target string, timeout time.Duration) (float64, error) {
	target = strings.TrimSpace(target)
	if target == "" || strings.HasPrefix(target, "-") || strings.ContainsAny(target, " \t\r\n") {
		return 0, fmt.Errorf("invalid ping target")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	waitSeconds := int(timeout.Round(time.Second) / time.Second)
	if waitSeconds < 1 {
		waitSeconds = 1
	}
	out, err := exec.CommandContext(ctx, "ping", "-c", "3", "-W", strconv.Itoa(waitSeconds), target).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ping %s: %w", target, err)
	}
	match := summaryPattern.FindStringSubmatch(string(out))
	if len(match) != 2 {
		return 0, fmt.Errorf("ping %s returned no average RTT", target)
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse ping RTT: %w", err)
	}
	return value, nil
}

func ParseAverage(output string) (float64, bool) {
	match := summaryPattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	return value, err == nil
}
