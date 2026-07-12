package apply

import (
	"os"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

func DNS(dns *config.DNS, hostname string, dryRun bool) error {
	if dns == nil {
		return nil
	}

	var lines []string
	lines = append(lines, "# routier:"+hostname)
	if len(dns.Search) > 0 {
		lines = append(lines, "search "+strings.Join(dns.Search, " "))
	}

	for _, ns := range dns.Nameservers {
		lines = append(lines, "nameserver "+ns)
	}

	lines = append(lines, "")
	content := strings.Join(lines, "\n")

	dest := "/etc/resolv.conf"
	if dryRun {
		log.Info().Str("dest", dest).Msg("would write")
		return nil
	}

	log.Info().Str("dest", dest).Msg("wrote")
	return os.WriteFile(dest, []byte(content), 0644)
}
