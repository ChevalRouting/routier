//go:build !linux

package unixauth

import "errors"

func Verify(user, password string) (bool, error) {
	return false, errors.New("unixauth: unsupported platform")
}

func Hash(password string) (string, error) {
	return "", errors.New("unixauth: unsupported platform")
}
