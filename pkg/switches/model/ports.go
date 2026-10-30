package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var rangePattern = regexp.MustCompile(`^(.*)\[(\d+)-(\d+)\](.*)$`)

func ExpandPorts(spec string) ([]string, error) {
	var ports []string
	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		expanded, err := expandItem(item)
		if err != nil {
			return nil, err
		}

		ports = append(ports, expanded...)
	}

	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports in %q", spec)
	}

	return ports, nil
}

func expandItem(item string) ([]string, error) {
	match := rangePattern.FindStringSubmatch(item)
	if match == nil {
		return []string{item}, nil
	}

	first, _ := strconv.Atoi(match[2])
	last, _ := strconv.Atoi(match[3])
	if last < first {
		return nil, fmt.Errorf("invalid range in %q", item)
	}

	ports := make([]string, 0, last-first+1)
	for n := first; n <= last; n++ {
		ports = append(ports, match[1]+strconv.Itoa(n)+match[4])
	}

	return ports, nil
}
