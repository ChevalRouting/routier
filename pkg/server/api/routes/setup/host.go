package setup

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	"github.com/ChevalRouting/routier/pkg/host/takeover"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const resolvConf = "/etc/resolv.conf"

// @Summary  Describe the host's current network for onboarding
// @Tags setup
// @Produce json
// @Success 200 {object} types.Response[types.SetupHost]
// @Security BearerAuth
// @Router /api/setup/host [get]
func Host(w http.ResponseWriter, _ *http.Request) {
	host, err := currentHost()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read host network"))
		return
	}

	host.Nameservers = nameservers(resolvConf)
	host.TakenOver = takeover.Done()
	types.OK(w, host)
}

func nameservers(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{}
	}

	defer func(action func() error) { _ = action() }(f.Close)
	servers := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "nameserver" {
			servers = append(servers, fields[1])
		}
	}

	return servers
}
