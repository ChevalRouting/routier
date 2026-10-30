package svc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func applyDNSVRFs(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	if !render.LocalDNSServerEnabled(cfg) && ServiceRunning("named") {
		content, err := os.ReadFile(render.NamedConfDest)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		if strings.HasPrefix(string(content), "// routier:") {
			if err := runCmdOutput([]string{"rc-service", "named", "stop"}, 10*time.Second); err != nil {
				return err
			}
		}
	}

	desired := map[string]*config.DNSServer{}
	if cfg.DNS != nil && cfg.DNS.Server != nil {
		desired = cfg.DNS.Server.VRFs
	}

	paths, err := filepath.Glob(render.DNSVRFRoot + "/*/run/launcher.pid")
	if err != nil {
		return err
	}

	for _, path := range paths {
		name := filepath.Base(filepath.Dir(filepath.Dir(path)))
		server := desired[name]
		if server == nil || !server.Enabled {
			if err := stopDNSVRF(path); err != nil {
				return err
			}
		}
	}

	names := []string{}
	for name, server := range desired {
		if server != nil && server.Enabled {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	for _, name := range names {
		if err := applyDNSVRF(name, desired[name]); err != nil {
			return fmt.Errorf("DNS in VRF %s: %w", name, err)
		}
	}

	return nil
}

func dnsVRFPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid < 2 {
		return 0, fmt.Errorf("invalid DNS pid file %s", path)
	}

	if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
		return 0, nil
	}

	data, err = os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return 0, err
	}

	conf := filepath.Join(filepath.Dir(filepath.Dir(path)), "named.conf")
	if !strings.Contains(string(data), "named\x00") || !strings.Contains(string(data), "-c\x00"+conf+"\x00") {
		return 0, fmt.Errorf("pid file %s does not identify its resolver", path)
	}

	return pid, nil
}

func stopDNSVRF(path string) error {
	pid, err := dnsVRFPID(path)
	if err != nil || pid == 0 {
		return err
	}

	return runCmdOutput([]string{"start-stop-daemon", "--stop", "--pidfile", path, "--retry", "TERM/5/KILL/1"}, 10*time.Second)
}

func applyDNSVRF(name string, server *config.DNSServer) error {
	directory := render.DNSVRFDirectory(name)
	for _, subdir := range []string{"run", "log", "data", "zones"} {
		if err := runCmdOutput([]string{"install", "-d", "-m", "0750", "-o", "named", "-g", "named", filepath.Join(directory, subdir)}, 0); err != nil {
			return err
		}
	}

	conf := filepath.Join(directory, "named.conf")
	if err := runCmdOutput([]string{"chown", "root:named", conf}, 0); err != nil {
		return err
	}

	if err := runCmdOutput([]string{"chmod", "0640", conf}, 0); err != nil {
		return err
	}

	if err := runCmdOutput([]string{"named-checkconf", "-z", conf}, 15*time.Second); err != nil {
		return err
	}

	path := filepath.Join(directory, "run/launcher.pid")
	pid, err := dnsVRFPID(path)
	if err != nil {
		return err
	}

	if pid > 0 {
		return syscall.Kill(pid, syscall.SIGHUP)
	}

	threads := server.Threads
	if threads == 0 {
		threads = 1
	}

	if err := os.Remove(filepath.Join(directory, "run/named.pid")); err != nil && !os.IsNotExist(err) {
		return err
	}

	args := []string{"start-stop-daemon", "--start", "--background", "--make-pidfile", "--pidfile", path, "--exec", "/sbin/ip", "--", "vrf", "exec", name, "/usr/sbin/named", "-f", "-u", "named", "-n", strconv.Itoa(threads), "-c", conf}
	if err := runCmdOutput(args, 10*time.Second); err != nil {
		return err
	}

	for attempt := 0; attempt < 30; attempt++ {
		if _, err := os.Stat(filepath.Join(directory, "run/named.pid")); err == nil {
			pid, err := dnsVRFPID(path)
			if err == nil && pid > 0 {
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("resolver did not start; inspect %s", filepath.Join(directory, "log/named.log"))
}
