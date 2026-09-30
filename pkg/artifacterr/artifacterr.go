package artifacterr

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

const (
	ToolNft            = "nft"
	ToolVtysh          = "vtysh"
	ToolRadvd          = "radvd"
	ToolKea            = "kea"
	ToolNamed          = "named"
	ToolNamedCheckzone = "named-checkzone"
)

var (
	nftLine   = regexp.MustCompile(`:(\d+):(\d+)(?:-\d+)?:\s*Error:\s*(.*)`)
	vtyshLine = regexp.MustCompile(`^line (\d+): (.*)$`)
	radvdLine = regexp.MustCompile(`(?::| line )(\d+)`)
	keaParen  = regexp.MustCompile(`\(([^()]*):(\d+):(\d+)\)`)
	keaLead   = regexp.MustCompile(`:(\d+):(\d+):\s*(.*)`)
	namedLine = regexp.MustCompile(`^(?:.*?):(\d+):\s*(?:error:\s*)?(.*)$`)
	zoneLine  = regexp.MustCompile(`^(?:.*?):(\d+):\s*(.*)$`)
)

func Parse(tool, dest, out string) []types.ArtifactError {
	var errs []types.ArtifactError
	switch tool {
	case ToolNft:
		errs = parseNft(out)
	case ToolVtysh:
		errs = parseVtysh(out)
	case ToolRadvd:
		errs = parseRadvd(out)
	case ToolKea:
		errs = parseKea(out)
	case ToolNamed:
		errs = parseNamed(out)
	case ToolNamedCheckzone:
		errs = parseZone(out)
	}

	if len(errs) == 0 {
		msg := strings.TrimSpace(out)
		if msg == "" {
			return nil
		}

		errs = []types.ArtifactError{{Message: msg}}
	}

	for i := range errs {
		errs[i].Tool = tool
		errs[i].Dest = dest
	}

	return errs
}

func parseNft(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		m := nftLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		ln, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		errs = append(errs, types.ArtifactError{Line: ln, Column: col, Message: strings.TrimSpace(m[3])})
	}

	return errs
}

func parseVtysh(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := vtyshLine.FindStringSubmatch(trimmed); m != nil {
			ln, _ := strconv.Atoi(m[1])
			errs = append(errs, types.ArtifactError{Line: ln, Message: strings.TrimSpace(m[2])})
			continue
		}

		if strings.HasPrefix(trimmed, "% ") {
			errs = append(errs, types.ArtifactError{Message: strings.TrimSpace(trimmed[2:])})
		}
	}

	return errs
}

func parseRadvd(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(strings.ToLower(trimmed), "error") && !strings.Contains(trimmed, "line ") {
			continue
		}

		ln := 0
		if m := radvdLine.FindStringSubmatch(trimmed); m != nil {
			ln, _ = strconv.Atoi(m[1])
		}

		errs = append(errs, types.ArtifactError{Line: ln, Message: trimmed})
	}

	return errs
}

func isWarning(line string) bool {
	lower := strings.ToLower(line)

	return strings.Contains(lower, "warning:") || strings.Contains(lower, " warning ")
}

func parseNamed(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isWarning(trimmed) {
			continue
		}

		if m := namedLine.FindStringSubmatch(trimmed); m != nil {
			ln, _ := strconv.Atoi(m[1])
			errs = append(errs, types.ArtifactError{Line: ln, Message: strings.TrimSpace(m[2])})
			continue
		}

		if idx := strings.Index(trimmed, "fatal error:"); idx >= 0 {
			errs = append(errs, types.ArtifactError{Message: strings.TrimSpace(trimmed[idx+len("fatal error:"):])})
		}
	}

	return errs
}

func parseZone(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isWarning(trimmed) || strings.HasSuffix(trimmed, "OK") {
			continue
		}

		if m := zoneLine.FindStringSubmatch(trimmed); m != nil {
			ln, _ := strconv.Atoi(m[1])
			errs = append(errs, types.ArtifactError{Line: ln, Message: strings.TrimSpace(m[2])})
			continue
		}

		if strings.Contains(trimmed, "not loaded") || strings.Contains(strings.ToLower(trimmed), "error") {
			errs = append(errs, types.ArtifactError{Message: trimmed})
		}
	}

	return errs
}

func parseKea(out string) []types.ArtifactError {
	var errs []types.ArtifactError
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := keaParen.FindStringSubmatch(trimmed); m != nil {
			ln, _ := strconv.Atoi(m[2])
			col, _ := strconv.Atoi(m[3])
			msg := strings.TrimSpace(keaParen.ReplaceAllString(trimmed, ""))
			errs = append(errs, types.ArtifactError{Line: ln, Column: col, Message: strings.TrimSpace(msg)})
			continue
		}

		if m := keaLead.FindStringSubmatch(trimmed); m != nil {
			ln, _ := strconv.Atoi(m[1])
			col, _ := strconv.Atoi(m[2])
			errs = append(errs, types.ArtifactError{Line: ln, Column: col, Message: strings.TrimSpace(m[3])})
		}
	}

	return errs
}
