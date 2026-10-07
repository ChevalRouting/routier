package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/host/motd"
	"github.com/ChevalRouting/routier/pkg/net/netlink"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newFirstbootCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "firstboot",
		Short:  "generate default config on first boot (no-op if config exists)",
		Hidden: true,
		RunE:   runFirstboot,
	}
}

const firstbootPass = "routier"

func runFirstboot(command *cobra.Command, _ []string) error {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		return nil
	}

	nics := usableNICs()
	if len(nics) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "routier firstboot: no usable ethernet interfaces found; skipping")
		return nil
	}

	hash, err := sha512Crypt(firstbootPass)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	cfg, err := config.Default()
	if err != nil {
		return fmt.Errorf("load default config: %w", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "routier"
	}

	cfg.Hostname = hostname

	ifaces := make(map[string]*config.Interface, len(nics))
	for _, nic := range nics {
		ifaces[nic] = &config.Interface{Select: nic, Addresses: []string{"dhcp"}}
	}

	cfg.Interfaces = ifaces

	if err := os.MkdirAll("/etc/routier", 0750); err != nil {
		return err
	}

	if err := os.MkdirAll("/var/lib/routier", 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(defaultConfigPath, data, 0640); err != nil {
		return err
	}

	cmd := exec.Command("chpasswd", "-e")
	cmd.Stdin = strings.NewReader("routier:" + hash + "\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chpasswd: %w", err)
	}

	if err := os.WriteFile("/var/lib/routier/ui-seed-password", []byte(firstbootPass+"\n"), 0600); err != nil {
		return fmt.Errorf("write ui-seed-password: %w", err)
	}

	motd.WriteIssue(VERSION)
	motd.Write(command.Context(), "routier", firstbootPass)

	_, _ = fmt.Printf("routier firstboot: generated %s (DHCP on: %s)\n", defaultConfigPath, strings.Join(nics, " "))
	return nil
}

func usableNICs() []string {
	nics, err := netlink.SystemNics()
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(nics))
	for _, n := range nics {
		names = append(names, n.Name)
	}

	return names
}

func sha512Crypt(pass string) (string, error) {
	out, err := exec.Command("openssl", "passwd", "-6", pass).Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
