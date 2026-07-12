package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/motd"
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

func runFirstboot(_ *cobra.Command, _ []string) error {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		return nil
	}

	nics := physicalNICs()
	if len(nics) == 0 {
		fmt.Fprintln(os.Stderr, "routier firstboot: no physical ethernet interfaces found; skipping")
		return nil
	}

	hash, err := sha512Crypt(firstbootPass)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "routier"
	}

	ifaces := make(map[string]*config.Interface, len(nics))
	for _, nic := range nics {
		ifaces[nic] = &config.Interface{Select: nic, Addresses: []string{"dhcp"}}
	}

	cfg := &config.Config{
		Version:    config.CurrentVersion,
		Hostname:   hostname,
		Interfaces: ifaces,
		Sysctl: map[string]string{
			"net.ipv4.ip_forward":          "1",
			"net.ipv6.conf.all.forwarding": "1",
		},
		Nftables: &config.NftablesConfig{
			Chains: map[string]*config.NftChain{
				"input": {Rules: `iifname "lo" accept
ct state established,related accept
ct state invalid drop
meta l4proto icmp accept
meta l4proto icmpv6 accept
tcp dport { ssh, 8080 } accept`},
				"forward": {Rules: `ct state established,related accept
ct state invalid drop`},
			},
		},
	}

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

	ver := VERSION
	issue := "\nWelcome to Routier"
	if ver != "" && ver != "dev" {
		issue += " " + ver
	}

	_ = os.WriteFile("/etc/issue", []byte(issue+"\n\n"), 0644)

	motd.Write("routier", firstbootPass)

	fmt.Printf("routier firstboot: generated %s (DHCP on: %s)\n", defaultConfigPath, strings.Join(nics, " "))
	return nil
}

func physicalNICs() []string {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil
	}

	var nics []string
	for _, e := range entries {
		name := e.Name()
		if name == "lo" {
			continue
		}

		if _, err := os.Stat("/sys/class/net/" + name + "/device"); os.IsNotExist(err) {
			continue
		}

		typ, _ := os.ReadFile("/sys/class/net/" + name + "/type")
		if strings.TrimSpace(string(typ)) != "1" {
			continue
		}

		switch {
		case strings.HasPrefix(name, "veth"),
			strings.HasPrefix(name, "virbr"),
			strings.HasPrefix(name, "docker"),
			strings.HasPrefix(name, "br-"),
			strings.HasPrefix(name, "bond"),
			strings.HasPrefix(name, "tap"),
			strings.HasPrefix(name, "tun"),
			strings.HasPrefix(name, "wg"):
			continue
		}

		nics = append(nics, name)
	}

	sort.Strings(nics)
	return nics
}

func sha512Crypt(pass string) (string, error) {
	out, err := exec.Command("openssl", "passwd", "-6", pass).Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
