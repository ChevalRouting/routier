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

	defer os.Remove(f.Name())

	if _, err := f.WriteString(ruleset); err != nil {
		f.Close()
		return "", err
	}

	f.Close()

	out, runErr := runCombined([]string{"nft", "-c", "-f", f.Name()}, 10*time.Second)
	return string(out), runErr
}
