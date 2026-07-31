package svc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/artifacterr"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
)

type artifactValidator struct {
	name   string
	tool   string
	bin    string
	prefix []string
	suffix []string
}

var artifactValidators = []artifactValidator{
	{name: "nftables/routier.nft", tool: artifacterr.ToolNft, bin: "nft", prefix: []string{"nft", "-c", "-f"}},
	{name: "frr/frr.conf", tool: artifacterr.ToolVtysh, bin: "vtysh", prefix: []string{"vtysh", "-C", "-f"}},
	{name: "radvd/radvd.conf", tool: artifacterr.ToolRadvd, bin: "radvd", prefix: []string{"radvd", "-C"}, suffix: []string{"-c"}},
	{name: "kea/kea-dhcp4.conf", tool: artifacterr.ToolKea, bin: "kea-dhcp4", prefix: []string{"kea-dhcp4", "-t"}},
	{name: "kea/kea-dhcp6.conf", tool: artifacterr.ToolKea, bin: "kea-dhcp6", prefix: []string{"kea-dhcp6", "-t"}},
}

func validatorFor(name string) *artifactValidator {
	for i := range artifactValidators {
		if artifactValidators[i].name == name {
			return &artifactValidators[i]
		}
	}

	return nil
}

func ValidateArtifacts(outputs []render.Output) []types.ArtifactError {
	dir, err := os.MkdirTemp("", "routier-check-")
	if err != nil {
		return []types.ArtifactError{{Message: "staging dir: " + err.Error()}}
	}

	defer os.RemoveAll(dir)

	var errs []types.ArtifactError
	for _, o := range outputs {
		v := validatorFor(o.Name)
		if v == nil {
			continue
		}

		if _, err := exec.LookPath(v.bin); err != nil {
			continue
		}

		f := filepath.Join(dir, filepath.Base(o.Dest))
		if err := os.WriteFile(f, []byte(o.Content), 0600); err != nil {
			errs = append(errs, types.ArtifactError{Tool: v.tool, Artifact: o.Name, Dest: o.Dest, Message: err.Error()})
			continue
		}

		argv := append(append([]string{}, v.prefix...), f)
		argv = append(argv, v.suffix...)

		out, runErr := runCombined(argv, 15*time.Second)
		if runErr == nil {
			continue
		}

		text := strings.TrimSpace(string(out))
		if text == "" {
			text = runErr.Error()
		}

		parsed := artifacterr.Parse(v.tool, o.Dest, text)
		for i := range parsed {
			parsed[i].Artifact = o.Name
		}

		errs = append(errs, parsed...)
	}

	return errs
}
