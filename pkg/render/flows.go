package render

import (
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

func renderFlowsTable(cfg *config.Config) string {
	if !cfg.Monitoring.FlowsEnabled() {
		return ""
	}

	devices := cfg.FlowDevices()
	if len(devices) == 0 {
		return ""
	}

	match := "iifname " + nftNameSet(devices)
	if rate := cfg.Monitoring.Flows.SamplingRate(); rate > 1 {
		match += fmt.Sprintf(" numgen random mod %d 0", rate)
	}

	rule := fmt.Sprintf("%s log group %d snaplen %d comment \"routier: sflow sampling\"", match, config.FlowLogGroup, config.FlowSnaplen)

	var b strings.Builder
	_, _ = b.WriteString("table inet routier_flows {\n")
	for _, hook := range []string{"forward", "input"} {
		_, _ = fmt.Fprintf(&b, "\tchain sample_%s {\n", hook)
		_, _ = fmt.Fprintf(&b, "\t\ttype filter hook %s priority filter - 10; policy accept;\n", hook)
		_, _ = fmt.Fprintf(&b, "\t\t%s\n", rule)
		_, _ = b.WriteString("\t}\n")
	}

	_, _ = b.WriteString("}\n")
	return b.String()
}

func nftNameSet(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}

	if len(quoted) == 1 {
		return quoted[0]
	}

	return "{ " + strings.Join(quoted, ", ") + " }"
}
