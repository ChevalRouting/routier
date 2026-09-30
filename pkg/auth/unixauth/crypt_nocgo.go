//go:build linux && !cgo

package unixauth

import "errors"

func cryptVerify(password, hash string) (bool, error) {
	return false, errors.New("unixauth: crypt unavailable (built without cgo)")
}

func cryptHash(password, setting string) (string, error) {
	return "", errors.New("unixauth: crypt unavailable (built without cgo)")
}
