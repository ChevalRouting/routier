//go:build linux

package unixauth

import (
	"bufio"
	"crypto/rand"
	"errors"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

var ErrNoShadow = errors.New("unixauth: user not present in /etc/shadow")

const saltAlphabet = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func Hash(password string) (string, error) {
	log.Info().Msg("starting Unix password hash")
	salt, err := randomSalt(16)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate Unix password salt")
		return "", err
	}

	hash, err := cryptHash(password, "$6$"+salt)
	if err != nil {
		log.Error().Err(err).Msg("Unix password hash failed")
		return "", err
	}

	if !strings.HasPrefix(hash, "$6$") {
		log.Error().Str("algorithm", cryptAlgorithm(hash)).Msg("crypt returned an unexpected password hash")
		return "", errors.New("unixauth: unexpected crypt output")
	}

	log.Info().Str("algorithm", cryptAlgorithm(hash)).Msg("Unix password hash completed")
	return hash, nil
}

func randomSalt(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	for i := range buf {
		buf[i] = saltAlphabet[int(buf[i])%len(saltAlphabet)]
	}

	return string(buf), nil
}

func Verify(user, password string) (bool, error) {
	if user == "" || user == "root" {
		log.Warn().Str("user", user).Msg("Unix password verification rejected reserved user")
		return false, nil
	}

	log.Info().Str("user", user).Msg("starting Unix password verification")
	hash, err := shadowHash(user)
	if err != nil {
		log.Error().Err(err).Str("user", user).Msg("failed to read Unix password hash")
		return false, err
	}

	if hash == "" || hash == "*" || strings.HasPrefix(hash, "!") {
		log.Warn().Str("user", user).Msg("Unix account has no usable password hash")
		return false, nil
	}

	if !strings.HasPrefix(hash, "$") {
		log.Warn().Str("user", user).Msg("Unix account uses an unsupported password hash format")
		return false, nil
	}

	verified, err := cryptVerify(password, hash)
	if err != nil {
		log.Error().Err(err).Str("user", user).Str("algorithm", cryptAlgorithm(hash)).Msg("Unix password verification failed")
		return false, err
	}

	log.Info().Bool("verified", verified).Str("user", user).Str("algorithm", cryptAlgorithm(hash)).Msg("Unix password verification completed")
	return verified, nil
}

func cryptAlgorithm(hash string) string {
	parts := strings.Split(hash, "$")
	if len(parts) < 2 || parts[1] == "" {
		return "unknown"
	}

	return parts[1]
}

func shadowHash(user string) (string, error) {
	f, err := os.Open("/etc/shadow")
	if err != nil {
		return "", err
	}

	defer func(action func() error) { _ = action() }(f.Close)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) >= 2 && fields[0] == user {
			return fields[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", ErrNoShadow
}
