package apply

import (
	"os"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
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

	if current, err := os.ReadFile(dest); err == nil && string(current) == content {
		log.Debug().Str("dest", dest).Msg("unchanged")
		return nil
	}

	log.Info().Str("dest", dest).Msg("wrote")
	return os.WriteFile(dest, []byte(content), 0644)
}

func PrepareDNSZoneSerials(cfg *config.Config, outputs []render.Output) error {
	for i := range outputs {
		output := &outputs[i]
		local, zone, scoped := render.DNSVRFZone(cfg, *output)
		if !scoped {
			local, zone = cfg, render.ZoneOriginFromName(output.Name)
		}

		if (!scoped && !strings.HasPrefix(output.Name, render.NamedZoneName)) || !render.AutomaticZoneSerial(local, zone) {
			continue
		}

		current, readErr := os.ReadFile(output.Dest)
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}

		output.Content = render.AdvanceZoneSerial(string(current), output.Content)
	}

	return nil
}
