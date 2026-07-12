package friends

import (
	"context"
	"fmt"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
)

func selfFingerprint(app *appctx.App) string {
	if app.Identity != nil {
		return app.Identity.Fingerprint()
	}

	return ""
}

func friendInfo(f *cfgpkg.Friend) types.FriendInfo {
	info := types.FriendInfo{
		Name:        f.Name,
		Hostname:    f.Hostname,
		URL:         f.URL,
		Enabled:     f.IsEnabled(),
		Fingerprint: f.Identity.Fingerprint,
		Paired:      f.Identity.X25519PublicKey != "",
	}

	if f.HA != nil {
		info.HA = f.HA.Enabled
	}

	return info
}

func pairFriend(ctx context.Context, app *appctx.App, cfg *cfgpkg.Config, f *cfgpkg.Friend) error {
	if app.Identity == nil {
		return fmt.Errorf("no local identity")
	}

	scheme := "http"
	if app.AdvertiseTLS {
		scheme = "https"
	}

	client := friendspkg.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	resp, err := client.Pair(ctx, app.Identity, cfg.Hostname, scheme, app.AdvertisePort)
	if err != nil {
		return err
	}

	if err := friendspkg.PinIdentity(f, resp.IdentityFingerprint, resp.IdentityPublicKey, resp.X25519PublicKey); err != nil {
		return err
	}

	if resp.Hostname != "" {
		f.Hostname = resp.Hostname
	}

	return nil
}

func formatHost(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}

	return host
}
