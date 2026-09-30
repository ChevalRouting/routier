//go:build linux && cgo

package unixauth

/*
#include <crypt.h>
#include <stdlib.h>
#include <string.h>

static int routier_crypt_verify(const char *key, const char *stored) {
	struct crypt_data data;
	memset(&data, 0, sizeof(data));
	char *res = crypt_r(key, stored, &data);
	if (res == NULL) {
		return -1;
	}
	return strcmp(res, stored) == 0 ? 1 : 0;
}

static int routier_crypt(const char *key, const char *setting, char *out, int outlen) {
	struct crypt_data data;
	memset(&data, 0, sizeof(data));
	char *res = crypt_r(key, setting, &data);
	if (res == NULL) {
		return -1;
	}
	strncpy(out, res, outlen - 1);
	out[outlen - 1] = 0;
	return 1;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/rs/zerolog/log"
)

func cryptVerify(password, hash string) (bool, error) {
	key := C.CString(password)
	defer C.free(unsafe.Pointer(key))
	stored := C.CString(hash)
	defer C.free(unsafe.Pointer(stored))

	log.Info().Str("algorithm", cryptAlgorithm(hash)).Msg("calling crypt_r to verify password")
	result := int(C.routier_crypt_verify(key, stored))
	if result < 0 {
		return false, fmt.Errorf("unixauth: crypt_r verify failed with status %d", result)
	}

	verified := result == 1
	log.Info().Bool("verified", verified).Str("algorithm", cryptAlgorithm(hash)).Msg("crypt_r password verification completed")

	return verified, nil
}

func cryptHash(password, setting string) (string, error) {
	key := C.CString(password)
	defer C.free(unsafe.Pointer(key))
	set := C.CString(setting)
	defer C.free(unsafe.Pointer(set))

	buf := (*C.char)(C.malloc(256))
	defer C.free(unsafe.Pointer(buf))

	log.Info().Str("algorithm", cryptAlgorithm(setting)).Msg("calling crypt_r to hash password")
	if C.routier_crypt(key, set, buf, 256) != 1 {
		return "", errors.New("unixauth: crypt failed")
	}

	hash := C.GoString(buf)
	log.Info().Str("algorithm", cryptAlgorithm(hash)).Msg("crypt_r password hashing completed")

	return hash, nil
}
