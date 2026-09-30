package failures

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var secretLine = regexp.MustCompile(`(?i)(private[_-]?key|preshared[_-]?key|password|secret|psk)(\s*[:=]\s*)(\S+)`)

func redactBytes(b []byte) []byte {
	return secretLine.ReplaceAll(b, []byte("${1}${2}REDACTED"))
}

func Export(id string, w io.Writer, redact bool) error {
	if !safeID(id) {
		return fmt.Errorf("invalid bundle id")
	}

	root := filepath.Join(baseDir, id)
	if _, err := os.Stat(root); err != nil {
		return err
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if redact {
			data = redactBytes(data)
		}

		rel, err := filepath.Rel(filepath.Dir(root), path)
		if err != nil {
			return err
		}

		hdr := &tar.Header{Name: filepath.ToSlash(rel), Mode: 0600, Size: int64(len(data)), ModTime: time.Now()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		_, err = tw.Write(data)
		return err
	})
	if walkErr != nil {
		_ = tw.Close()
		_ = gz.Close()
		return walkErr
	}

	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return err
	}

	return gz.Close()
}
