package setup

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Nics godoc
// @Summary  List physical system NICs
// @Tags setup
// @Produce json
// @Success 200 {object} types.Response[[]types.SystemNic]
// @Security BearerAuth
// @Router /api/system/nics [get]
func Nics(w http.ResponseWriter, _ *http.Request) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read interfaces"))
		return
	}

	nics := []types.SystemNic{}
	for _, e := range entries {
		name := e.Name()
		if name == "lo" {
			continue
		}

		base := filepath.Join("/sys/class/net", name)
		if _, err := os.Stat(filepath.Join(base, "device")); err != nil {
			continue
		}

		if strings.TrimSpace(readSysFile(filepath.Join(base, "type"))) != "1" {
			continue
		}

		switch {
		case strings.HasPrefix(name, "veth"), strings.HasPrefix(name, "virbr"),
			strings.HasPrefix(name, "docker"), strings.HasPrefix(name, "br-"),
			strings.HasPrefix(name, "bond"), strings.HasPrefix(name, "tap"),
			strings.HasPrefix(name, "tun"), strings.HasPrefix(name, "wg"):
			continue
		}

		nics = append(nics, types.SystemNic{
			Name:      name,
			MAC:       strings.TrimSpace(readSysFile(filepath.Join(base, "address"))),
			Operstate: strings.TrimSpace(readSysFile(filepath.Join(base, "operstate"))),
			Addrs:     nicAddrs(name),
		})
	}

	sort.Slice(nics, func(i, j int) bool { return nics[i].Name < nics[j].Name })
	types.OK(w, nics)
}

func readSysFile(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}
