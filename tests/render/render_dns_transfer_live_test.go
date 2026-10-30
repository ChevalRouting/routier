package rendertest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

const transferTestSecret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestDNSLiveTransfers(t *testing.T) {
	if os.Getenv("ROUTIER_TEST_DNS_LIVE") != "1" {
		t.Skip("requires isolated loopback addresses 127.0.0.20 and 127.0.0.21")
	}

	if _, err := exec.LookPath("named"); err != nil {
		t.Skip("named not installed")
	}

	if _, err := exec.LookPath("dig"); err != nil {
		t.Skip("dig not installed")
	}

	key := &config.DNSTransferKey{Name: "transfer.example.net", Secret: transferTestSecret}
	primary := config.DNSZone{Name: "example.net", Nameservers: []string{"ns.example.net"}, SOA: &config.DNSSOA{Email: "hostmaster@example.net", Serial: 1}, Records: []config.DNSRecord{{Name: "ns", Type: "A", Value: "192.0.2.1"}}, Secondaries: []string{"127.0.0.21", "127.0.0.22"}, TransferKey: key, NotifySource: "127.0.0.20"}
	command, zonePath := startTransferTestNamed(t, "127.0.0.20", primary)
	waitTransferTestAnswer(t, "127.0.0.20", "192.0.2.1")

	for _, source := range []string{"127.0.0.21", "127.0.0.22"} {
		answer := transferTestDig(t, "-b", source, "-y", "hmac-sha256:transfer.example.net:"+transferTestSecret, "@127.0.0.20", "example.net", "AXFR")
		if !strings.Contains(answer, "192.0.2.1") {
			t.Fatalf("authorized source %s refused: %s", source, answer)
		}
	}

	for _, args := range [][]string{
		{"-b", "127.0.0.21", "@127.0.0.20", "example.net", "AXFR"},
		{"-b", "127.0.0.23", "-y", "hmac-sha256:transfer.example.net:" + transferTestSecret, "@127.0.0.20", "example.net", "AXFR"},
		{"-b", "127.0.0.21", "-y", "hmac-sha256:transfer.example.net:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=", "@127.0.0.20", "example.net", "AXFR"},
	} {
		if answer := transferTestDig(t, args...); strings.Contains(answer, "192.0.2.1") {
			t.Fatalf("unauthorized transfer accepted: %s", answer)
		}
	}

	secondary := config.DNSZone{Name: "example.net", Primaries: []string{"127.0.0.20"}, TransferKey: key, TransferSource: "127.0.0.21"}
	startTransferTestNamed(t, "127.0.0.21", secondary)
	waitTransferTestAnswer(t, "127.0.0.21", "192.0.2.1")

	current, err := os.ReadFile(zonePath)
	if err != nil {
		t.Fatal(err)
	}

	next := render.AdvanceZoneSerial(string(current), strings.ReplaceAll(string(current), "192.0.2.1", "192.0.2.2"))
	if err := os.WriteFile(zonePath, []byte(next), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := command.Process.Signal(syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}

	waitTransferTestAnswer(t, "127.0.0.21", "192.0.2.2")
}

func transferTestDig(t *testing.T, args ...string) string {
	t.Helper()
	args = append([]string{"+time=1", "+tries=1"}, args...)
	output, _ := exec.Command("dig", args...).CombinedOutput()
	return string(output)
}

func waitTransferTestAnswer(t *testing.T, address, value string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var answer string
	for time.Now().Before(deadline) {
		answer = transferTestDig(t, "@"+address, "ns.example.net", "A", "+short")
		if strings.TrimSpace(answer) == value {
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("server %s did not serve %s: %s", address, value, transferTestDig(t, "@"+address, "ns.example.net", "A"))
}

func startTransferTestNamed(t *testing.T, address string, zone config.DNSZone) (*exec.Cmd, string) {
	t.Helper()
	dir := t.TempDir()
	outputs, err := render.All(&config.Config{Hostname: "transfer-test", DNS: &config.DNS{Server: &config.DNSServer{Enabled: true, Mode: "authoritative", Listen: []string{address}, AllowFrom: []string{"127.0.0.0/8"}, Zones: []config.DNSZone{zone}}}})
	if err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(dir, "rndc.key")
	if err := os.WriteFile(keyPath, []byte(`key "rndc-key" { algorithm hmac-sha256; secret "`+transferTestSecret+`"; };`), 0o600); err != nil {
		t.Fatal(err)
	}

	confPath := filepath.Join(dir, "named.conf")
	zonePath := filepath.Join(dir, "example.net.zone")
	for _, output := range outputs {
		if output.Name == render.NamedConfName {
			content := strings.ReplaceAll(output.Content, "/etc/bind/rndc.key", keyPath)
			content = strings.ReplaceAll(content, "/etc/bind/zones/", dir+"/")
			content = strings.ReplaceAll(content, "/var/bind/", dir+"/")
			content = strings.ReplaceAll(content, `directory "/var/bind"`, `directory "`+dir+`"`)
			content = strings.ReplaceAll(content, "/run/named/named.pid", filepath.Join(dir, "named.pid"))
			content = strings.ReplaceAll(content, "/var/log/named/named.log", filepath.Join(dir, "named.log"))
			content = strings.ReplaceAll(content, "inet 127.0.0.1 port", "inet "+address+" port")
			if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		if strings.HasPrefix(output.Name, render.NamedZoneName) {
			if err := os.WriteFile(zonePath, []byte(output.Content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	command := exec.Command("named", "-g", "-n", "1", "-c", confPath)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	return command, zonePath
}
