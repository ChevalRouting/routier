package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/host/live"
	"github.com/ChevalRouting/routier/pkg/host/motd"
	"github.com/spf13/cobra"
	vnl "github.com/vishvananda/netlink"
)

const (
	liveSeedPath = "/var/lib/routier/ui-seed-password"
	liveTLSCert  = "/var/lib/routier/tls.crt"
	liveTLSKey   = "/var/lib/routier/tls.key"
	liveUIPort   = 8080
)

func newLiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "live",
		Short:  "prepare a live boot: admin login, TLS certificate and console banner",
		Hidden: true,
		RunE:   runLive,
	}
	cmd.Flags().Duration("wait", 20*time.Second, "how long to wait for a network address")

	return cmd
}

func runLive(cmd *cobra.Command, _ []string) error {
	if !live.Booted(live.MountsPath) {
		return fmt.Errorf("not a live system")
	}

	if err := os.MkdirAll("/var/lib/routier", 0700); err != nil {
		return err
	}

	created, err := ensureLiveUser()
	if err != nil {
		return err
	}

	if _, err := os.Stat(liveSeedPath); created || err == nil {
		if err := resetLivePassword(); err != nil {
			return err
		}
	}

	hostname, _ := os.Hostname()
	if _, err := live.EnsureCert(liveTLSCert, liveTLSKey, hostname); err != nil {
		return fmt.Errorf("tls certificate: %w", err)
	}

	wait, _ := cmd.Flags().GetDuration("wait")
	writeLiveBanner(waitForAddresses(wait))
	return nil
}

func ensureLiveUser() (bool, error) {
	if exec.Command("id", "routier").Run() == nil {
		return false, nil
	}

	if out, err := exec.Command("adduser", "-D", "-s", "/bin/ash", "-h", "/home/routier", "routier").CombinedOutput(); err != nil {
		return false, fmt.Errorf("adduser: %w: %s", err, out)
	}

	_ = exec.Command("addgroup", "routier", "wheel").Run()
	return true, nil
}

func resetLivePassword() error {
	password, err := live.Password(16)
	if err != nil {
		return err
	}

	chpasswd := exec.Command("chpasswd")
	chpasswd.Stdin = strings.NewReader("routier:" + password + "\n")
	if out, err := chpasswd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd: %w: %s", err, out)
	}

	return os.WriteFile(liveSeedPath, []byte(password+"\n"), 0600)
}

func writeLiveBanner(addrs []string) {
	urls := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		urls = append(urls, fmt.Sprintf("https://%s:%d", addr, liveUIPort))
	}

	password := ""
	if data, err := os.ReadFile(liveSeedPath); err == nil {
		password = strings.TrimSpace(string(data))
	}

	header := motd.RenderIssue(VERSION)
	_ = os.WriteFile("/etc/motd", []byte(live.Banner(header, urls, password)), 0644)
	_ = os.WriteFile("/etc/issue", []byte(strings.ReplaceAll(header, `\`, `\\`)), 0644)
}

func waitForAddresses(wait time.Duration) []string {
	deadline := time.Now().Add(wait)
	for {
		addrs := globalIPv4()
		if len(addrs) > 0 || time.Now().After(deadline) {
			return addrs
		}

		time.Sleep(time.Second)
	}
}

func globalIPv4() []string {
	list, err := vnl.AddrList(nil, vnl.FAMILY_V4)
	if err != nil {
		return nil
	}

	var out []string
	for _, a := range list {
		if a.IP.IsGlobalUnicast() {
			out = append(out, a.IP.String())
		}
	}

	return out
}
