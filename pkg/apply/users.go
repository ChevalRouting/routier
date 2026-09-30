package apply

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const managedGecos = "managed-routier"

func Users(users map[string]*config.User, dryRun bool) error {
	for name, u := range users {
		if name == "routier" {
			if err := writeSSHKeys(name, u, dryRun); err != nil {
				return fmt.Errorf("user %s ssh_keys: %w", name, err)
			}

			continue
		}

		if err := ensureUser(name, u, dryRun); err != nil {
			return fmt.Errorf("user %s: %w", name, err)
		}

		if err := setPassword(name, u, dryRun); err != nil {
			return fmt.Errorf("user %s password: %w", name, err)
		}

		if err := writeSSHKeys(name, u, dryRun); err != nil {
			return fmt.Errorf("user %s ssh_keys: %w", name, err)
		}
	}

	return pruneUsers(users, dryRun)
}

func setPassword(name string, u *config.User, dryRun bool) error {
	if u.PasswordHash == "" {
		return nil
	}

	if current, err := shadowHash(name); err == nil && current == u.PasswordHash {
		return nil
	}

	if dryRun {
		log.Info().Str("user", name).Msg("would set password")
		return nil
	}

	cmd := exec.Command("chpasswd", "-e")
	cmd.Stdin = strings.NewReader(name + ":" + u.PasswordHash + "\n")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Info().Str("user", name).Msg("set password")
	return nil
}

func shadowHash(name string) (string, error) {
	f, err := os.Open("/etc/shadow")
	if err != nil {
		return "", err
	}

	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), ":", 3)
		if len(fields) >= 2 && fields[0] == name {
			return fields[1], nil
		}
	}

	return "", fmt.Errorf("user %s not in shadow", name)
}

func ensureUser(name string, u *config.User, dryRun bool) error {
	if userExists(name) {
		return syncGroups(name, u, dryRun)
	}

	shell := u.Shell
	if shell == "" {
		shell = "/bin/ash"
	}

	home := u.Home
	if home == "" {
		home = "/home/" + name
	}

	args := []string{"adduser", "-D", "-g", managedGecos}
	if u.System {
		args = append(args, "-S")
	}

	if u.UID > 0 {
		args = append(args, "-u", fmt.Sprintf("%d", u.UID))
	}

	args = append(args, "-s", shell, "-h", home, name)

	if dryRun {
		log.Info().Strs("cmd", args).Msg("would run")
		return nil
	}

	log.Info().Str("user", name).Msg("adduser")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if fi, err := os.Stat("/etc/routier"); err == nil && fi.IsDir() {
		if err := exec.Command("chown", "-R", name+":"+name, "/etc/routier").Run(); err != nil {
			log.Warn().Err(err).Str("user", name).Msg("chown /etc/routier")
		}
	}

	return syncGroups(name, u, dryRun)
}

func pruneUsers(want map[string]*config.User, dryRun bool) error {
	managed, err := managedUsers()
	if err != nil {
		return fmt.Errorf("pruneUsers: %w", err)
	}

	for _, name := range managed {
		if _, ok := want[name]; ok {
			continue
		}

		if dryRun {
			log.Info().Str("user", name).Msg("would deluser")
			continue
		}

		log.Info().Str("user", name).Msg("deluser")
		cmd := exec.Command("deluser", name)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("deluser %s: %w", name, err)
		}
	}

	return nil
}

func managedUsers() ([]string, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.SplitN(line, ":", 7)
		if len(fields) >= 5 && fields[4] == managedGecos {
			names = append(names, fields[0])
		}
	}

	return names, scanner.Err()
}

func syncGroups(name string, u *config.User, dryRun bool) error {
	for _, g := range u.Groups {
		if userInGroup(name, g) {
			continue
		}

		args := []string{"addgroup", name, g}
		if dryRun {
			log.Info().Strs("cmd", args).Msg("would run")
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Warn().Err(err).Str("user", name).Str("group", g).Msg("addgroup failed")
		}
	}

	return nil
}

func userInGroup(user, group string) bool {
	f, err := os.Open("/etc/group")
	if err != nil {
		return false
	}

	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), ":", 4)
		if len(fields) < 4 || fields[0] != group {
			continue
		}

		for _, m := range strings.Split(fields[3], ",") {
			if strings.TrimSpace(m) == user {
				return true
			}
		}
	}

	return false
}

func writeSSHKeys(name string, u *config.User, dryRun bool) error {
	home := u.Home
	if home == "" {
		home = "/home/" + name
	}

	sshDir := filepath.Join(home, ".ssh")
	authFile := filepath.Join(sshDir, "authorized_keys")
	if len(u.SSHKeys) == 0 {
		return removeSSHKeys(name, authFile, dryRun)
	}

	content := strings.Join(u.SSHKeys, "\n") + "\n"
	existing, readErr := os.ReadFile(authFile)
	contentChanged := readErr != nil || string(existing) != content

	if dryRun {
		if contentChanged {
			log.Info().Str("file", authFile).Int("keys", len(u.SSHKeys)).Msg("would write ssh keys")
		}

		return nil
	}

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(sshDir, 0700); err != nil {
		return err
	}

	if contentChanged {
		if err := os.WriteFile(authFile, []byte(content), 0600); err != nil {
			return err
		}
	}
	if err := os.Chmod(authFile, 0600); err != nil {
		return err
	}

	if out, err := exec.Command("chown", name+":"+name, sshDir, authFile).CombinedOutput(); err != nil {
		return fmt.Errorf("chown: %w: %s", err, strings.TrimSpace(string(out)))
	}

	log.Info().Str("file", authFile).Int("keys", len(u.SSHKeys)).Msg("wrote ssh keys")
	return nil
}

func removeSSHKeys(name, authFile string, dryRun bool) error {
	if _, err := os.Stat(authFile); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if dryRun {
		log.Info().Str("user", name).Str("file", authFile).Msg("would remove ssh keys")
		return nil
	}

	if err := os.Remove(authFile); err != nil {
		return err
	}

	log.Info().Str("user", name).Str("file", authFile).Msg("removed ssh keys")
	return nil
}

func userExists(name string) bool {
	err := exec.Command("id", name).Run()
	return err == nil
}
