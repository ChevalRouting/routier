package config

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/artifacterr"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// GetNftablesVars godoc
// @Summary  nftables variables for the staged config
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[[]render.NftVar]
// @Security BearerAuth
// @Router /api/config/nftables/vars [get]
func GetNftablesVars(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	cfgpkg.ResolveInterfaces(cfg)
	types.OK(w, render.NftVars(cfg, render.WithFriends(friendcache.InterpolationVars())))
}

var nftChainOpen = regexp.MustCompile(`^chain (\w+) \{$`)

// ValidateNftables godoc
// @Summary  Validate a draft nftables ruleset with `nft -c`
// @Tags config
// @Produce json
// @Param body body config.NftablesConfig true "draft nftables section"
// @Success 200 {object} types.Response[types.NftValidateResult]
// @Security BearerAuth
// @Router /api/config/nftables/validate [post]
func ValidateNftables(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read request body").Write(w)
		return
	}

	var draft cfgpkg.NftablesConfig
	if err := json.Unmarshal(body, &draft); err != nil {
		types.Error(log.Logger, w, types.Errorf(http.StatusBadRequest, "invalid nftables data: %v", err))
		return
	}

	cfg.Nftables = &draft
	cfgpkg.ResolveInterfaces(cfg)

	outputs, err := render.All(cfg, render.WithFriends(friendcache.InterpolationVars()))
	if err != nil {
		types.OK(w, types.NftValidateResult{Errors: []types.NftValidateError{{Message: err.Error()}}})
		return
	}

	var ruleset string
	for _, o := range outputs {
		if o.Dest == "/etc/nftables.d/routier.nft" {
			ruleset = o.Content
			break
		}
	}

	out, checkErr := svc.NftCheck(ruleset)
	if checkErr == nil {
		types.OK(w, types.NftValidateResult{OK: true})
		return
	}

	errs := parseNftErrors(out)
	locateNftErrors(errs, ruleset, draft)
	types.OK(w, types.NftValidateResult{Errors: errs})
}

func parseNftErrors(out string) []types.NftValidateError {
	parsed := artifacterr.Parse(artifacterr.ToolNft, "", out)
	errs := make([]types.NftValidateError, len(parsed))
	for i, e := range parsed {
		errs[i] = types.NftValidateError{Line: e.Line, Message: e.Message}
	}

	return errs
}

type nftRef struct {
	section string
	line    int
}

func locateNftErrors(errs []types.NftValidateError, ruleset string, draft cfgpkg.NftablesConfig) {
	refs := buildNftRefs(ruleset, draft)
	for i := range errs {
		if errs[i].Line < 1 || errs[i].Line >= len(refs) {
			continue
		}

		if r := refs[errs[i].Line]; r.section != "" {
			errs[i].Section = r.section
			errs[i].Line = r.line
		}
	}
}

func buildNftRefs(ruleset string, draft cfgpkg.NftablesConfig) []nftRef {
	lines := strings.Split(ruleset, "\n")
	refs := make([]nftRef, len(lines)+1)

	tableAt := len(lines)
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "table inet routier") {
			tableAt = i
			break
		}
	}

	mapUserBlock(refs, lines[:tableAt], "defines", draft.Defines)

	chain := ""
	inUser, sawAuto, afterPolicy := false, false, false
	userIdx, managedCount := 0, 0
	for i := tableAt; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if m := nftChainOpen.FindStringSubmatch(line); m != nil {
			chain, inUser, sawAuto, afterPolicy, userIdx = m[1], false, false, false, 0
			managedCount = draftManagedCount(draft, chain)
			continue
		}

		if chain == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "type "):
			afterPolicy = true
		case strings.Contains(line, "routier auto rules"):
			sawAuto, inUser = true, false
		case strings.Contains(line, "user rules"):
			inUser, userIdx = true, 0
		case line == "}":
			chain = ""
		default:
			if !inUser && !sawAuto && afterPolicy && line != "" {
				inUser = true
			}

			if inUser {
				userIdx++
				if userIdx > managedCount {
					refs[i+1] = nftRef{section: chain, line: userIdx - managedCount}
				}
			}
		}
	}

	return refs
}

func draftManagedCount(draft cfgpkg.NftablesConfig, chain string) int {
	ch := draft.Chains[chain]
	if ch == nil {
		return 0
	}

	n := 0
	for _, r := range ch.Managed {
		if !r.Disabled {
			n++
		}
	}

	return n
}

func mapUserBlock(refs []nftRef, region []string, section, block string) {
	block = strings.TrimRight(block, "\n")
	if block == "" {
		return
	}

	want := strings.Split(block, "\n")
	for start := 0; start+len(want) <= len(region); start++ {
		match := true
		for j := range want {
			if strings.TrimSpace(region[start+j]) != strings.TrimSpace(want[j]) {
				match = false
				break
			}
		}

		if match {
			for j := range want {
				refs[start+j+1] = nftRef{section: section, line: j + 1}
			}

			return
		}
	}
}
