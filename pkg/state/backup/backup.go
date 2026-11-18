package backup

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const userTemplatesDir = "/etc/routier/templates"

func (r Restored) Load() (*config.Config, error) {
	if r.Config == nil {
		return config.LoadAndValidate(r.ConfigPath, true)
	}

	return config.LoadBytesAndValidate(r.ConfigPath, r.Config, true)
}

type Restored struct {
	ConfigPath string
	Config     []byte
}

type manifest struct {
	Version    string    `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	ConfigPath string    `json:"config_path"`
}

func Create(outputPath, configPath string, cfg *config.Config) error {
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	removeOnErr := true
	defer func() { closeArchive(outputPath, out, removeOnErr) }()

	if err := Write(out, configPath, cfg); err != nil {
		return err
	}

	log.Info().Str("output", outputPath).Msg("backup created")
	removeOnErr = false
	return nil
}

func Write(w io.Writer, configPath string, cfg *config.Config) error {
	files, err := collectFiles(configPath, cfg)
	if err != nil {
		return err
	}

	zstdCmd := exec.Command("zstd", "-T0", "-c")
	zstdCmd.Stdout = w

	pr, pw := io.Pipe()
	zstdCmd.Stdin = pr

	if err := zstdCmd.Start(); err != nil {
		return fmt.Errorf("zstd: %w", err)
	}

	tw := tar.NewWriter(pw)

	absConfig, _ := filepath.Abs(configPath)
	m := manifest{
		Version:    "1",
		CreatedAt:  time.Now().UTC(),
		ConfigPath: absConfig,
	}

	if err := writeJSON(tw, "manifest.json", m); err != nil {
		_ = pw.CloseWithError(err)
		_ = zstdCmd.Wait()
		return err
	}

	for _, path := range files {
		if err := writeFile(tw, path); err != nil {
			_ = pw.CloseWithError(err)
			_ = zstdCmd.Wait()
			return fmt.Errorf("backup %s: %w", path, err)
		}

		log.Debug().Str("file", path).Msg("backed up")
	}

	_ = tw.Close()
	_ = pw.Close()

	if err := zstdCmd.Wait(); err != nil {
		return fmt.Errorf("zstd: %w", err)
	}

	log.Debug().Int("files", len(files)).Msg("backup written")
	return nil
}

func Restore(archivePath, configPath string) (Restored, error) {
	root, err := restoreRoot(configPath)
	if err != nil {
		return Restored{}, err
	}

	zstdCmd := exec.Command("zstd", "-d", "-c", archivePath)
	stdout, err := zstdCmd.StdoutPipe()
	if err != nil {
		return Restored{}, err
	}

	if err := zstdCmd.Start(); err != nil {
		return Restored{}, fmt.Errorf("zstd: %w", err)
	}

	tr := tar.NewReader(stdout)
	var m *manifest
	var count int
	var restored Restored

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			_ = zstdCmd.Wait()
			return Restored{}, fmt.Errorf("reading archive: %w", err)
		}

		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				_ = zstdCmd.Wait()
				return Restored{}, fmt.Errorf("reading manifest: %w", err)
			}

			m = &manifest{}
			if err := json.Unmarshal(data, m); err != nil {
				_ = zstdCmd.Wait()
				return Restored{}, fmt.Errorf("parsing manifest: %w", err)
			}

			continue
		}

		dest, err := safePath(hdr.Name)
		if err != nil {
			_ = zstdCmd.Wait()
			return Restored{}, fmt.Errorf("unsafe path %q: %w", hdr.Name, err)
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		if m != nil && dest == filepath.Clean(m.ConfigPath) {
			data, err := io.ReadAll(tr)
			if err != nil {
				_ = zstdCmd.Wait()
				return Restored{}, fmt.Errorf("reading %s: %w", dest, err)
			}

			restored.Config = data
			continue
		}

		if !within(root, dest) {
			log.Warn().Str("file", dest).Str("root", root).Msg("restore: skipping file outside the config directory")
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			_ = zstdCmd.Wait()
			return Restored{}, err
		}

		f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode)&0777)
		if err != nil {
			_ = zstdCmd.Wait()
			return Restored{}, err
		}

		_, copyErr := io.Copy(f, tr)
		_ = f.Close()
		if copyErr != nil {
			_ = zstdCmd.Wait()
			return Restored{}, fmt.Errorf("extracting %s: %w", dest, copyErr)
		}

		log.Debug().Str("file", dest).Msg("restored")
		count++
	}

	if err := zstdCmd.Wait(); err != nil {
		return Restored{}, fmt.Errorf("zstd: %w", err)
	}

	if m == nil {
		return Restored{}, fmt.Errorf("archive has no manifest.json")
	}

	log.Info().Int("files", count).Msg("archive extracted")
	restored.ConfigPath = configPath
	return restored, nil
}

func restoreRoot(configPath string) (string, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	return filepath.Dir(abs), nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func collectFiles(configPath string, cfg *config.Config) ([]string, error) {
	seen := map[string]bool{}
	var paths []string

	add := func(p string) { addBackupPath(seen, &paths, p) }

	resolve := func(p string) string { return resolveBackupPath(cfg, p) }

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(absConfig); err != nil {
		return nil, err
	}

	add(absConfig)

	if cfg.Nftables != nil {
		for _, ch := range cfg.Nftables.Chains {
			for _, f := range ch.Files {
				add(resolve(f))
			}
		}

		for _, f := range cfg.Nftables.Include {
			add(resolve(f))
		}
	}

	for _, wg := range cfg.VPN.Wireguard {
		if wg.PrivateKeyFile != "" {
			add(resolve(wg.PrivateKeyFile))
		}

		for _, peer := range wg.Peers {
			if peer.PresharedKeyFile != "" {
				add(resolve(peer.PresharedKeyFile))
			}
		}
	}

	_ = filepath.WalkDir(userTemplatesDir, func(path string, d fs.DirEntry, err error) error { return visitBackupFile(add, path, d, err) })

	return paths, nil
}

func writeJSON(tw *tar.Writer, name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}); err != nil {
		return err
	}

	_, err = tw.Write(data)
	return err
}

func writeFile(tw *tar.Writer, absPath string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(f.Close)

	info, err := f.Stat()
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    strings.TrimPrefix(absPath, "/"),
		Mode:    int64(info.Mode()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}

func safePath(name string) (string, error) {
	cleaned := filepath.Clean("/" + name)
	if !strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path escapes root")
	}

	return cleaned, nil
}

func closeArchive(outputPath string, out *os.File, removeOnErr bool) {
	_ = out.Close()
	if removeOnErr {
		_ = os.Remove(outputPath)
	}
}

func addBackupPath(seen map[string]bool, paths *[]string, p string) {
	abs := filepath.Clean(p)
	if seen[abs] {
		return
	}

	if _, err := os.Stat(abs); err != nil {
		log.Warn().Str("file", abs).Msg("backup: skipping missing file")
		return
	}

	seen[abs] = true
	*paths = append(*paths, abs)
}

func resolveBackupPath(cfg *config.Config, p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(cfg.BaseDir, p)
}

func visitBackupFile(add func(p string), path string, d fs.DirEntry, err error) error {
	if err != nil {
		log.Warn().Err(err).Str("file", path).Msg("backup: skipping unreadable file")
		return nil
	}

	if d.IsDir() {
		return nil
	}

	add(path)
	return nil
}
