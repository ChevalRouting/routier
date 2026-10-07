package app

import "github.com/golang-jwt/jwt/v5"

func JWTKeyFunc(secret []byte) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) { return jWTKeyFuncCallback(secret, t) }
}

func jWTKeyFuncCallback(secret []byte, t *jwt.Token) (any, error) {
	if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	return secret, nil
}
