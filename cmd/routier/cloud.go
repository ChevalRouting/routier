package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/cibi"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/host/cloud"
	"github.com/ChevalRouting/routier/pkg/host/live"
	"github.com/ChevalRouting/routier/pkg/host/takeover"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	cloudSeedPath = "/var/lib/routier/ui-seed-password"
	cloudDBPath   = "/var/lib/routier/web.db"
)

func newCloudCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "cloud",
		Short:  "provision this instance from the cloud-init routier section (once per instance)",
		Hidden: true,
		RunE:   runCloud,
	}
}

func runCloud(cmd *cobra.Command, _ []string) error {
	hostname, _ := os.Hostname()
	created, err := live.EnsureCert(liveTLSCert, liveTLSKey, hostname)
	if err != nil {
		return fmt.Errorf("tls certificate: %w", err)
	}

	if created && exec.Command("rc-service", "routier-ui", "status").Run() == nil {
		_ = exec.Command("rc-service", "routier-ui", "restart").Run()
	}

	id := cloud.InstanceID(cloud.InstanceIDPath)
	if id == "" {
		return fmt.Errorf("no cloud-init instance id at %s", cloud.InstanceIDPath)
	}

	if cloud.Provisioned(cloud.StampPath, id) {
		return nil
	}

	prov, err := cloud.Load(cloud.CloudConfigPath)
	if err != nil {
		consolef("routier: ignoring the routier section of the user-data: %v", err)
	}

	if err := provisionPassword(prov.Password); err != nil {
		return err
	}

	if prov.Config != nil {
		if err := provisionConfig(cmd.Context(), prov.Config); err != nil {
			consolef("routier: provisioning the configuration failed: %v", err)
		}
	}

	if prov.Cibi != nil {
		provisionCibi(*prov.Cibi)
	}

	return cloud.MarkProvisioned(cloud.StampPath, id)
}

func provisionCibi(join cibi.Join) {
	if err := cibi.SavePending(filepath.Join(filepath.Dir(cloudDBPath), cibi.PendingFilename), join); err != nil {
		consolef("routier: ignoring routier.cibi in the user-data: %v", err)
		return
	}

	if exec.Command("rc-service", "routier-ui", "status").Run() == nil {
		_ = exec.Command("rc-service", "routier-ui", "restart").Run()
	}

	consolef("routier: joining cibi at %s once the web server is up", join.Server)
}

func provisionPassword(password string) error {
	generated := password == ""
	if generated {
		var err error
		if password, err = live.Password(16); err != nil {
			return err
		}
	}

	chpasswd := exec.Command("chpasswd")
	chpasswd.Stdin = strings.NewReader("routier:" + password + "\n")
	if out, err := chpasswd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd: %w: %s", err, out)
	}

	if !generated {
		_ = os.Remove(cloudSeedPath)
		return nil
	}

	if err := os.WriteFile(cloudSeedPath, []byte(password+"\n"), 0600); err != nil {
		return err
	}

	consolef("routier: web UI login routier / %s (one-time password, changed at first login)", password)
	return nil
}

func provisionConfig(ctx context.Context, data []byte) error {
	if err := os.MkdirAll("/etc/routier", 0750); err != nil {
		return err
	}

	if err := os.WriteFile(defaultConfigPath, data, 0640); err != nil {
		return err
	}

	cfg, err := config.LoadAndValidate(defaultConfigPath, true)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if _, err := takeover.Run(); err != nil {
		return fmt.Errorf("take over networking: %w", err)
	}

	if err := cloud.DisableNetworkConfig(cloud.NetworkDropIn); err != nil {
		log.Warn().Err(err).Msg("cloud-init network rendering left enabled")
	}

	if err := managers.Boot(ctx, cfg, friends.LoadCacheVars(defaultFriendsCache)); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	markOnboardingComplete(ctx)
	consolef("routier: configuration from the user-data applied")
	return nil
}

func markOnboardingComplete(ctx context.Context) {
	db, err := webdb.InitDB(ctx, cloudDBPath)
	if err != nil {
		log.Warn().Err(err).Msg("onboarding left pending")
		return
	}

	defer func(action func() error) { _ = action() }(db.Close)
	if err := webdb.SetSetting(ctx, db, webdb.SettingOnboardingComplete, "true"); err != nil {
		log.Warn().Err(err).Msg("onboarding left pending")
	}
}

func consolef(format string, args ...any) {
	line := fmt.Sprintf(format, args...) + "\n"
	_, _ = fmt.Fprint(os.Stderr, line)
	if console, err := os.OpenFile("/dev/console", os.O_WRONLY, 0); err == nil {
		_, _ = console.WriteString(line)
		_ = console.Close()
	}
}
