package svc

import (
	"os"
	"time"
)

func NftCheck(ruleset string) (string, error) {
	f, err := os.CreateTemp("", "routier-nftcheck-*.nft")
	if err != nil {
		return "", err
	}

	defer func(path string) { _ = os.Remove(path) }(f.Name())

	if _, err := f.WriteString(ruleset); err != nil {
		_ = f.Close()
		return "", err
	}

	_ = f.Close()

	out, runErr := runCombined([]string{"nft", "-c", "-f", f.Name()}, 10*time.Second)
	return string(out), runErr
}
