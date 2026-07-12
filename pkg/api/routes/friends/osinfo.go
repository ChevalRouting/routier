package friends

import (
	"os"
	"strings"
)

func osInfo() string {
	distro := ""
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
				distro = strings.Trim(strings.TrimSpace(v), `"`)
				break
			}
		}
	}

	kernel := ""
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		kernel = strings.TrimSpace(string(data))
	}

	switch {
	case distro != "" && kernel != "":
		return distro + " (kernel " + kernel + ")"
	case distro != "":
		return distro
	default:
		return kernel
	}
}
